package filestore

import (
	"context"
	"io"
	"time"

	"store"
)

// Repository provides a developer-friendly facade for file operations.
// It wraps a low-level FileStore and exposes consistent pagination types.
type Repository struct {
	store FileStore
}

// NewRepository creates a new files repository backed by the given FileStore.
func NewRepository(fs FileStore) *Repository { return &Repository{store: fs} }

// Save stores content from an io.Reader with the provided name and content type.
// Returns the generated file ID and resolved metadata.
func (r *Repository) Save(ctx context.Context, name string, reader io.Reader, contentType string) (FileID, *FileMetadata, error) {
	f := &file{metadata: FileMetadata{Name: name, Path: name, Size: 0, ContentType: contentType}, stream: io.NopCloser(reader)}
	return r.store.Store(ctx, f)
}

// SaveBytes stores an in-memory byte slice.
func (r *Repository) SaveBytes(ctx context.Context, name string, content []byte, contentType string) (FileID, *FileMetadata, error) {
	return r.Save(ctx, name, bytesReader(content), contentType)
}

// SavePath stores a local file from disk.
func (r *Repository) SavePath(ctx context.Context, path string) (FileID, *FileMetadata, error) {
	f, err := FileFromLocalPath(path)
	if err != nil {
		return InvalidFileID, nil, err
	}
	return r.store.Store(ctx, f)
}

// Get retrieves a file stream and its metadata.
func (r *Repository) Get(ctx context.Context, id FileID) (io.ReadCloser, *FileMetadata, error) {
	f, err := r.store.Retrieve(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	rc, err := f.Stream()
	if err != nil {
		return nil, nil, err
	}
	md := f.Metadata()
	return rc, &md, nil
}

// Delete removes a file by ID.
func (r *Repository) Delete(ctx context.Context, id FileID) error {
	return r.store.Delete(ctx, id)
}

// List returns file metadata using store cursor params.
// Note: Underlying adapters may not return encoded cursors; NextCursor will be the adapter token.
func (r *Repository) List(ctx context.Context, params store.CursorParams) (store.CursorResult[FileMetadata], error) {
	items, nextToken, err := r.store.List(ctx, params.PageSize, params.Cursor)
	if err != nil {
		return store.CursorResult[FileMetadata]{}, err
	}
	res := store.CursorResult[FileMetadata]{
		Items:      items,
		NextCursor: nextToken,
		HasMore:    int32(len(items)) == params.PageSize,
		TotalCount: -1,
	}
	return res, nil
}

// URL returns a public URL for the file (if available).
func (r *Repository) URL(ctx context.Context, id FileID) (string, error) {
	return r.store.GetURL(ctx, id)
}

// PresignedURL returns a temporary signed URL when supported.
func (r *Repository) PresignedURL(ctx context.Context, id FileID, expiration time.Duration) (string, error) {
	return r.store.GeneratePresignedURL(ctx, id, expiration)
}

// Helper: lightweight bytes reader without extra allocations.
func bytesReader(b []byte) io.Reader { return (*sliceReader)(&b) }

type sliceReader []byte

func (s *sliceReader) Read(p []byte) (int, error) {
	if len(*s) == 0 {
		return 0, io.EOF
	}
	n := copy(p, *s)
	*s = (*s)[n:]
	return n, nil
}
