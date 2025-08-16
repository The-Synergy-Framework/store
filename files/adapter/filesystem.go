package adapter

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	filestore "store/files"
	"strconv"
	"strings"
	"time"

	"core/validation"
)

// FilesystemConfig configures the filesystem filestore.
type FilesystemConfig struct {
	Root        string `validate:"required"`
	BaseURL     string `validate:"omitempty"`
	SecretKey   string `validate:"omitempty"`
	MaxFileSize int64  `validate:"min:0"` // 0 = unlimited
	ChunkSize   int    `validate:"min:0"` // bytes per write; default 2MB if 0
}

// Validate validates the filesystem configuration.
func (c FilesystemConfig) Validate() error {
	res := validation.Validate(c)
	if res != nil && !res.IsValid {
		msgs := make([]string, 0, len(res.Errors))
		for _, e := range res.Errors {
			msgs = append(msgs, e.Error())
		}
		return fmt.Errorf("invalid filesystem config: %s", strings.Join(msgs, "; "))
	}
	// If BaseURL is set (used for public URLs and presigned URLs), SecretKey should be provided
	if strings.TrimSpace(c.BaseURL) != "" && strings.TrimSpace(c.SecretKey) == "" {
		return fmt.Errorf("SecretKey is required when BaseURL is set")
	}
	return nil
}

// filesystemAdapter implements filestore.FileStore directly.
type filesystemAdapter struct {
	root        string
	baseURL     string
	secretKey   string
	maxSize     int64
	chunkSize   int
	httpHandler http.Handler
}

// NewFilesystem creates a filesystem filestore from config.
func NewFilesystem(cfg FilesystemConfig) (filestore.FileStore, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	ad := &filesystemAdapter{
		root:      cfg.Root,
		baseURL:   cfg.BaseURL,
		secretKey: cfg.SecretKey,
		maxSize:   cfg.MaxFileSize,
		chunkSize: cfg.ChunkSize,
	}
	if ad.chunkSize <= 0 {
		ad.chunkSize = 2 * 1024 * 1024 // 2MB default
	}
	if cfg.BaseURL != "" {
		ad.httpHandler = http.StripPrefix("/files/", http.FileServer(http.Dir(cfg.Root)))
	}
	return ad, nil
}

// FileStore interface implementation
func (a *filesystemAdapter) Store(ctx context.Context, f filestore.File) (filestore.FileID, *filestore.FileMetadata, error) {
	md := f.Metadata()
	stream, err := f.Stream()
	if err != nil {
		return filestore.InvalidFileID, nil, err
	}
	defer stream.Close()

	// Prepare hasher and temp file in target shard directory
	h := sha256.New()
	// Write to a shard temp location to allow atomic rename
	// Compute a temporary path under root to avoid loading everything in memory
	tmpDir := a.root
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return filestore.InvalidFileID, nil, err
	}
	tmpFile, err := os.CreateTemp(tmpDir, "upload-*")
	if err != nil {
		return filestore.InvalidFileID, nil, err
	}
	defer func() { _ = tmpFile.Close(); _ = os.Remove(tmpFile.Name()) }()

	// Stream copy: read chunks, update hash and write
	var written int64
	bufSize := a.chunkSize
	buf := make([]byte, bufSize)
	for {
		if a.maxSize > 0 && written >= a.maxSize {
			return filestore.InvalidFileID, nil, fmt.Errorf("file exceeds max size: %d", a.maxSize)
		}
		n, rerr := stream.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			_, herr := h.Write(chunk)
			if herr != nil {
				return filestore.InvalidFileID, nil, herr
			}
			if _, werr := tmpFile.Write(chunk); werr != nil {
				return filestore.InvalidFileID, nil, werr
			}
			written += int64(n)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return filestore.InvalidFileID, nil, rerr
		}
	}
	// Derive content hash and final ID (contentHash + original name)
	contentHash := hex.EncodeToString(h.Sum(nil))
	h2 := sha256.New()
	h2.Write([]byte(fmt.Sprintf("%s:%s", contentHash, md.Name)))
	finalHash := hex.EncodeToString(h2.Sum(nil))
	id := filestore.FileID(finalHash[:filestore.FileIDLength])

	// Compute final path with sharding and ensure directory exists
	finalPath := a.pathFor(id)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return filestore.InvalidFileID, nil, err
	}
	// If file already exists (dedup), discard temp and return metadata
	exists, err := a.Exists(ctx, id)
	if err != nil {
		return filestore.InvalidFileID, nil, err
	}
	if exists {
		meta, err := a.GetMetadata(ctx, id)
		return id, meta, err
	}
	// Sync temp to disk before rename (best-effort)
	_ = tmpFile.Sync()
	if err := tmpFile.Close(); err != nil {
		return filestore.InvalidFileID, nil, err
	}
	if err := os.Rename(tmpFile.Name(), finalPath); err != nil {
		return filestore.InvalidFileID, nil, err
	}
	meta, err := a.GetMetadata(ctx, id)
	return id, meta, err
}

func (a *filesystemAdapter) Retrieve(ctx context.Context, id filestore.FileID) (filestore.File, error) {
	p := a.pathFor(id)
	stream, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(p)
	if err != nil {
		stream.Close()
		return nil, err
	}
	name := filestore.ExtractOriginalFileName(id)
	if name == "" {
		name = string(id)
	}
	ext := filepath.Ext(name)
	md := filestore.FileMetadata{Name: name, Path: string(id), Size: info.Size(), ContentType: mime.TypeByExtension(ext)}
	return &fileAdapter{metadata: md, stream: stream}, nil
}

func (a *filesystemAdapter) Delete(ctx context.Context, id filestore.FileID) error {
	return os.Remove(a.pathFor(id))
}

func (a *filesystemAdapter) Exists(ctx context.Context, id filestore.FileID) (bool, error) {
	_, err := os.Stat(a.pathFor(id))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (a *filesystemAdapter) GetMetadata(ctx context.Context, id filestore.FileID) (*filestore.FileMetadata, error) {
	p := a.pathFor(id)
	info, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	name := filestore.ExtractOriginalFileName(id)
	if name == "" {
		name = string(id)
	}
	ext := filepath.Ext(name)
	md := filestore.FileMetadata{
		Name:        name,
		Path:        string(id),
		Size:        info.Size(),
		ContentType: mime.TypeByExtension(ext),
	}
	return &md, nil
}

func (a *filesystemAdapter) List(ctx context.Context, pageSize int32, pageToken string) ([]filestore.FileMetadata, string, error) {
	// Traverse sharded directories and collect names; for very large trees, prefer an index.
	var names []string
	err := filepath.WalkDir(a.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Only include leaf files (skip temp files)
		if strings.HasPrefix(filepath.Base(path), "upload-") {
			return nil
		}
		rel, _ := filepath.Rel(a.root, path)
		parts := strings.Split(rel, string(filepath.Separator))
		name := parts[len(parts)-1]
		names = append(names, name)
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	sort.Strings(names)
	start := 0
	if pageToken != "" {
		for i, n := range names {
			if n == pageToken {
				start = i + 1
				break
			}
		}
	}
	end := start + int(pageSize)
	if end > len(names) {
		end = len(names)
	}
	nextToken := ""
	if end < len(names) {
		nextToken = names[end-1]
	}

	items := make([]filestore.FileMetadata, 0, end-start)
	for _, n := range names[start:end] {
		id := filestore.FileID(n)
		md, err := a.GetMetadata(ctx, id)
		if err != nil {
			return nil, "", err
		}
		items = append(items, *md)
	}
	return items, nextToken, nil
}

func (a *filesystemAdapter) GeneratePresignedURL(ctx context.Context, id filestore.FileID, expires time.Duration) (string, error) {
	if a.baseURL == "" {
		return "", fmt.Errorf("base URL not configured for presigned URLs")
	}
	exists, err := a.Exists(ctx, id)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", os.ErrNotExist
	}
	token := a.generateToken(id, expires)
	return fmt.Sprintf("%s/files/%s?token=%s", strings.TrimSuffix(a.baseURL, "/"), id, token), nil
}

func (a *filesystemAdapter) GetURL(ctx context.Context, id filestore.FileID) (string, error) {
	if a.baseURL == "" {
		return "file://" + a.pathFor(id), nil
	}
	return fmt.Sprintf("%s/files/%s", strings.TrimSuffix(a.baseURL, "/"), id), nil
}

// Helper methods
func (a *filesystemAdapter) shardPath(id filestore.FileID) string {
	name := string(id)
	if len(name) < 4 {
		return a.root
	}
	return filepath.Join(a.root, name[0:2], name[2:4])
}

func (a *filesystemAdapter) pathFor(id filestore.FileID) string {
	return filepath.Join(a.shardPath(id), string(id))
}

func (a *filesystemAdapter) generateToken(fileID filestore.FileID, expires time.Duration) string {
	expiresAt := time.Now().Add(expires)
	ts := strconv.FormatInt(expiresAt.Unix(), 10)
	sig := a.generateSignature(string(fileID), ts)
	return fmt.Sprintf("%s.%s", ts, sig)
}

func (a *filesystemAdapter) generateSignature(path, timestamp string) string {
	data := fmt.Sprintf("%s:%s", path, timestamp)
	h := hmac.New(sha256.New, []byte(a.secretKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// fileAdapter implements filestore.File
type fileAdapter struct {
	metadata filestore.FileMetadata
	stream   io.ReadCloser
}

func (f *fileAdapter) Metadata() filestore.FileMetadata { return f.metadata }
func (f *fileAdapter) Stream() (io.ReadCloser, error)   { return f.stream, nil }

// Open creates a filesystem filestore from config (convenience alias for NewFilesystem).
func Open(cfg FilesystemConfig) (filestore.FileStore, error) { return NewFilesystem(cfg) }
