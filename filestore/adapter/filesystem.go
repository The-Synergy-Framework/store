package adapter

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"store/filestore"
)

// FilesystemConfig configures the filesystem filestore.
type FilesystemConfig struct {
	Root      string
	BaseURL   string
	SecretKey string
}

// filesystemAdapter implements filestore.FileStore directly.
type filesystemAdapter struct {
	root        string
	baseURL     string
	secretKey   string
	httpHandler http.Handler
}

// NewFilesystem creates a filesystem filestore from config.
func NewFilesystem(cfg FilesystemConfig) filestore.FileStore {
	ad := &filesystemAdapter{
		root:      cfg.Root,
		baseURL:   cfg.BaseURL,
		secretKey: cfg.SecretKey,
	}
	if cfg.BaseURL != "" {
		ad.httpHandler = http.StripPrefix("/files/", http.FileServer(http.Dir(cfg.Root)))
	}
	return ad
}

// FileStore interface implementation
func (a *filesystemAdapter) Store(ctx context.Context, f filestore.File) (filestore.FileID, error) {
	md := f.Metadata()
	stream, err := f.Stream()
	if err != nil {
		return filestore.InvalidFileID, err
	}
	defer stream.Close()
	id, err := filestore.GenerateFileIDFromStream(stream, md.Name)
	if err != nil {
		return filestore.InvalidFileID, err
	}
	exists, err := a.Exists(ctx, id)
	if err != nil {
		return filestore.InvalidFileID, err
	}
	if exists {
		return id, nil
	}
	if err := os.MkdirAll(a.root, 0755); err != nil {
		return filestore.InvalidFileID, err
	}
	w, err := f.Stream()
	if err != nil {
		return filestore.InvalidFileID, err
	}
	defer w.Close()
	dst, err := os.Create(a.pathFor(id))
	if err != nil {
		return filestore.InvalidFileID, err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, w); err != nil {
		return filestore.InvalidFileID, err
	}
	return id, nil
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
	return err == nil, err
}

func (a *filesystemAdapter) GetPresignedURL(ctx context.Context, id filestore.FileID, expires time.Duration) (string, error) {
	if a.baseURL == "" {
		return "", fmt.Errorf("base URL not configured for presigned URLs")
	}
	exists, err := a.Exists(ctx, id)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", sql.ErrNoRows
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
func (a *filesystemAdapter) pathFor(id filestore.FileID) string {
	return filepath.Join(a.root, string(id))
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

// Open creates a new filesystem filestore with the given configuration.
func Open(cfg FilesystemConfig) filestore.FileStore {
	return NewFilesystem(cfg)
}
