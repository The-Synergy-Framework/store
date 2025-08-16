package adapter

import (
	"context"
	"time"

	"store/filestore"
)

// Adapter defines a backend for the filestore (e.g., filesystem, s3, ipfs).
// It mirrors the filestore.FileStore interface.
type Adapter interface {
	Name() string

	Store(ctx context.Context, file filestore.File) (filestore.FileID, error)
	Retrieve(ctx context.Context, fileID filestore.FileID) (filestore.File, error)
	Delete(ctx context.Context, fileID filestore.FileID) error
	Exists(ctx context.Context, fileID filestore.FileID) (bool, error)
	GetPresignedURL(ctx context.Context, fileID filestore.FileID, expires time.Duration) (string, error)
	GetURL(ctx context.Context, fileID filestore.FileID) (string, error)

	Close() error
}

// Factory creates an adapter from a configuration.
type Factory func(config interface{}) (Adapter, error)

// registry is a simple in-memory adapter registry.
var registry = map[string]Factory{}

// Register adds an adapter factory by name.
func Register(name string, factory Factory) {
	registry[name] = factory
}

// Get returns an adapter factory by name.
func Get(name string) (Factory, bool) {
	f, ok := registry[name]
	return f, ok
}

// List returns the names of registered adapters.
func List() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}
