package filestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"time"
)

type FileID string

const (
	FileIDLength  = 16
	InvalidFileID = FileID("")
)

type FileMetadata struct {
	Name        string
	Path        string
	Size        int64
	ContentType string
}

type File interface {
	Metadata() FileMetadata
	Stream() (io.ReadCloser, error)
}

type file struct {
	metadata FileMetadata
	stream   io.ReadCloser
}

func (f *file) Metadata() FileMetadata         { return f.metadata }
func (f *file) Stream() (io.ReadCloser, error) { return f.stream, nil }

func FileFromLocalPath(path string) (File, error) {
	stream, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	metadata := FileMetadata{
		Name:        filepath.Base(path),
		Path:        path,
		Size:        info.Size(),
		ContentType: mime.TypeByExtension(filepath.Ext(path)),
	}
	return &file{metadata: metadata, stream: stream}, nil
}

func GenerateFileID(content []byte, originalName string) FileID {
	data := fmt.Sprintf("%s:%s", hex.EncodeToString(content), originalName)
	h := sha256.New()
	h.Write([]byte(data))
	hash := hex.EncodeToString(h.Sum(nil))
	return FileID(hash[:FileIDLength])
}

func GenerateFileIDFromStream(stream io.Reader, originalName string) (FileID, error) {
	h := sha256.New()
	_, err := io.Copy(h, stream)
	if err != nil {
		return InvalidFileID, err
	}
	contentHash := hex.EncodeToString(h.Sum(nil))
	data := fmt.Sprintf("%s:%s", contentHash, originalName)
	h.Reset()
	h.Write([]byte(data))
	finalHash := hex.EncodeToString(h.Sum(nil))
	return FileID(finalHash[:FileIDLength]), nil
}

func ExtractOriginalFileName(fileID FileID) string { return "" }

// FileStore interface

type FileStore interface {
	Store(ctx context.Context, file File) (FileID, error)
	Retrieve(ctx context.Context, fileID FileID) (File, error)
	Delete(ctx context.Context, fileID FileID) error
	Exists(ctx context.Context, fileID FileID) (bool, error)
	GetPresignedURL(ctx context.Context, fileID FileID, expires time.Duration) (string, error)
	GetURL(ctx context.Context, fileID FileID) (string, error)
}
