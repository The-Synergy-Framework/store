package files

import (
	"context"
	"io"
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

// FileStore defines the interface for file storage operations.
type FileStore interface {
	// Store saves a file and returns its ID and metadata
	Store(ctx context.Context, file File) (FileID, *FileMetadata, error)

	// Retrieve gets a file by ID
	Retrieve(ctx context.Context, id FileID) (File, error)

	// Delete removes a file by ID
	Delete(ctx context.Context, id FileID) error

	// Exists checks if a file exists
	Exists(ctx context.Context, id FileID) (bool, error)

	// GetMetadata returns file metadata without the content
	GetMetadata(ctx context.Context, id FileID) (*FileMetadata, error)

	// List returns files with pagination
	List(ctx context.Context, pageSize int32, pageToken string) ([]FileMetadata, string, error)

	// GeneratePresignedURL creates a temporary URL for file access (if supported)
	GeneratePresignedURL(ctx context.Context, id FileID, expiration time.Duration) (string, error)

	// GetURL returns the URL for a file
	GetURL(ctx context.Context, id FileID) (string, error)
}
