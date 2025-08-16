package filesystem

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

type Store struct {
	root        string
	baseURL     string
	secretKey   string
	httpHandler http.Handler
}

func New(root, baseURL, secretKey string) *Store {
	s := &Store{root: root, baseURL: baseURL, secretKey: secretKey}
	if baseURL != "" {
		s.httpHandler = http.StripPrefix("/files/", http.FileServer(http.Dir(root)))
	}
	return s
}

func (s *Store) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.httpHandler == nil {
		http.Error(w, "File serving not configured", http.StatusServiceUnavailable)
		return
	}
	if token := r.URL.Query().Get("token"); token != "" {
		if !s.validateToken(r.URL.Path, token) {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}
	}
	s.httpHandler.ServeHTTP(w, r)
}

func (s *Store) validateToken(path, token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	ts, sig := parts[0], parts[1]
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	if time.Now().After(time.Unix(tsInt, 0)) {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(s.generateSignature(path, ts)))
}

func (s *Store) generateSignature(path, timestamp string) string {
	data := fmt.Sprintf("%s:%s", path, timestamp)
	h := hmac.New(sha256.New, []byte(s.secretKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *Store) generateToken(fileID filestore.FileID, expires time.Duration) string {
	expiresAt := time.Now().Add(expires)
	ts := strconv.FormatInt(expiresAt.Unix(), 10)
	sig := s.generateSignature(string(fileID), ts)
	return fmt.Sprintf("%s.%s", ts, sig)
}

func (s *Store) pathFor(id filestore.FileID) string { return filepath.Join(s.root, string(id)) }

func (s *Store) Store(ctx context.Context, f filestore.File) (filestore.FileID, error) {
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
	exists, err := s.Exists(ctx, id)
	if err != nil {
		return filestore.InvalidFileID, err
	}
	if exists {
		return id, nil
	}
	if err := os.MkdirAll(s.root, 0755); err != nil {
		return filestore.InvalidFileID, err
	}
	w, err := f.Stream()
	if err != nil {
		return filestore.InvalidFileID, err
	}
	defer w.Close()
	dst, err := os.Create(s.pathFor(id))
	if err != nil {
		return filestore.InvalidFileID, err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, w); err != nil {
		return filestore.InvalidFileID, err
	}
	return id, nil
}

func (s *Store) Retrieve(ctx context.Context, id filestore.FileID) (filestore.File, error) {
	p := s.pathFor(id)
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

type fileAdapter struct {
	metadata filestore.FileMetadata
	stream   io.ReadCloser
}

func (f *fileAdapter) Metadata() filestore.FileMetadata { return f.metadata }
func (f *fileAdapter) Stream() (io.ReadCloser, error)   { return f.stream, nil }

func (s *Store) Delete(ctx context.Context, id filestore.FileID) error {
	return os.Remove(s.pathFor(id))
}
func (s *Store) Exists(ctx context.Context, id filestore.FileID) (bool, error) {
	_, err := os.Stat(s.pathFor(id))
	return err == nil, err
}

func (s *Store) GetPresignedURL(ctx context.Context, id filestore.FileID, expires time.Duration) (string, error) {
	if s.baseURL == "" {
		return "", fmt.Errorf("base URL not configured for presigned URLs")
	}
	exists, err := s.Exists(ctx, id)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", sql.ErrNoRows
	}
	token := s.generateToken(id, expires)
	return fmt.Sprintf("%s/files/%s?token=%s", strings.TrimSuffix(s.baseURL, "/"), id, token), nil
}

func (s *Store) GetURL(ctx context.Context, id filestore.FileID) (string, error) {
	if s.baseURL == "" {
		return "file://" + s.pathFor(id), nil
	}
	return fmt.Sprintf("%s/files/%s", strings.TrimSuffix(s.baseURL, "/"), id), nil
}
