package adapter

import (
	"context"
	"fmt"
	"time"

	"store/filestore"
)

// Service wraps an Adapter and exposes the filestore.FileStore interface.
type Service struct {
	ad Adapter
}

// NewService creates a new filestore service from an adapter.
func NewService(ad Adapter) *Service { return &Service{ad: ad} }

// OpenWithName opens a filestore by adapter name using the registry.
func OpenWithName(_ context.Context, name string, config interface{}) (*Service, error) {
	factory, ok := Get(name)
	if !ok {
		return nil, fmt.Errorf("filestore adapter not found: %s", name)
	}
	ad, err := factory(config)
	if err != nil {
		return nil, err
	}
	return NewService(ad), nil
}

// Delegate methods
func (s *Service) Store(ctx context.Context, file filestore.File) (filestore.FileID, error) {
	return s.ad.Store(ctx, file)
}

func (s *Service) Retrieve(ctx context.Context, fileID filestore.FileID) (filestore.File, error) {
	return s.ad.Retrieve(ctx, fileID)
}

func (s *Service) Delete(ctx context.Context, fileID filestore.FileID) error {
	return s.ad.Delete(ctx, fileID)
}

func (s *Service) Exists(ctx context.Context, fileID filestore.FileID) (bool, error) {
	return s.ad.Exists(ctx, fileID)
}

func (s *Service) GetPresignedURL(ctx context.Context, fileID filestore.FileID, expires time.Duration) (string, error) {
	return s.ad.GetPresignedURL(ctx, fileID, expires)
}

func (s *Service) GetURL(ctx context.Context, fileID filestore.FileID) (string, error) {
	return s.ad.GetURL(ctx, fileID)
}

func (s *Service) Close() error { return s.ad.Close() }
