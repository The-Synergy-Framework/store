package kvstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"core/entity"
	"store"
	"store/kv/adapter"
)

// Service wraps a KV adapter and provides the key-value service interface.
// This follows the guard service pattern and extends the shared service base.
type Service struct {
	adapter    adapter.Adapter
	connection adapter.Connection
	config     *adapter.Config
}

// Ensure Service implements the service interface.
var _ store.Service = (*Service)(nil)

// NewService creates a new KV service with the given adapter.
func NewService(adpt adapter.Adapter, config *adapter.Config) *Service {
	return &Service{
		adapter: adpt,
		config:  config,
	}
}

// Connect establishes the key-value store connection.
func (s *Service) Connect(ctx context.Context) error {
	connection, err := s.adapter.Connect(ctx, s.config)
	if err != nil {
		return store.WrapConnectionError(err, "connect", s.adapter.Name(), s.config.Host)
	}

	// Test connection
	pingCtx := ctx
	var cancel context.CancelFunc
	if s.config.ConnectTimeout > 0 {
		pingCtx, cancel = context.WithTimeout(ctx, s.config.ConnectTimeout)
		defer cancel()
	}

	if err := connection.Ping(pingCtx); err != nil {
		_ = connection.Close()
		return store.WrapConnectionError(err, "ping", s.adapter.Name(), s.config.Host)
	}

	s.connection = connection
	return nil
}

// Connection returns the underlying connection.
func (s *Service) Connection() adapter.Connection {
	return s.connection
}

// Adapter returns the underlying adapter.
func (s *Service) Adapter() adapter.Adapter {
	return s.adapter
}

// Close closes the connection.
func (s *Service) Close() error {
	if s.connection != nil {
		return s.connection.Close()
	}
	return nil
}

// Stats returns connection statistics.
func (s *Service) Stats() interface{} {
	if s.connection != nil {
		return s.connection.Stats()
	}
	return nil
}

// NewRepository creates a new repository for the given entity type.
func (s *Service) NewRepository(entity entity.Entity) store.Repository {
	return NewRepository(s, entity)
}

// Repository creates a new repository for the given entity type (alias for NewRepository).
func (s *Service) Repository(entity entity.Entity) *Repository {
	return NewRepository(s, entity)
}

// WithTimeout creates a context with timeout for operations.
func (s *Service) WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// Basic KV operations

// Get retrieves a value by key.
func (s *Service) Get(ctx context.Context, key string) ([]byte, error) {
	return s.connection.Get(ctx, key)
}

// Set stores a value with optional expiration.
func (s *Service) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	return s.connection.Set(ctx, key, value, expiration)
}

// Delete removes a key.
func (s *Service) Delete(ctx context.Context, key string) error {
	return s.connection.Delete(ctx, key)
}

// Exists checks if a key exists.
func (s *Service) Exists(ctx context.Context, key string) (bool, error) {
	return s.connection.Exists(ctx, key)
}

// JSON operations for entities

// GetJSON retrieves and unmarshals a JSON value.
func (s *Service) GetJSON(ctx context.Context, key string, target interface{}) error {
	data, err := s.connection.Get(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, target)
}

// SetJSON marshals and stores a JSON value.
func (s *Service) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return s.connection.Set(ctx, key, data, expiration)
}

// Batch operations

// MGet retrieves multiple values.
func (s *Service) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	return s.connection.MGet(ctx, keys)
}

// MSet stores multiple values.
func (s *Service) MSet(ctx context.Context, pairs map[string][]byte, expiration time.Duration) error {
	return s.connection.MSet(ctx, pairs, expiration)
}

// MDelete removes multiple keys.
func (s *Service) MDelete(ctx context.Context, keys []string) error {
	return s.connection.MDelete(ctx, keys)
}

// Pattern operations

// Keys returns all keys matching a pattern.
func (s *Service) Keys(ctx context.Context, pattern string) ([]string, error) {
	return s.connection.Keys(ctx, pattern)
}

// Scan returns keys matching a pattern with pagination.
func (s *Service) Scan(ctx context.Context, cursor string, pattern string, count int) ([]string, string, error) {
	return s.connection.Scan(ctx, cursor, pattern, count)
}

// ScanWithPagination returns keys with standard pagination.
func (s *Service) ScanWithPagination(ctx context.Context, pattern string, pageSize int32, cursor string) ([]string, string, error) {
	// Use the new cursor-based pagination
	paginator := store.NewPaginator()
	params := paginator.ParseParams(pageSize, cursor)

	return s.connection.Scan(ctx, cursor, pattern, int(params.PageSize))
}

// Expiration operations

// Expire sets expiration for a key.
func (s *Service) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return s.connection.Expire(ctx, key, expiration)
}

// TTL returns time-to-live for a key.
func (s *Service) TTL(ctx context.Context, key string) (time.Duration, error) {
	return s.connection.TTL(ctx, key)
}

// Atomic operations

// Incr increments a key by 1.
func (s *Service) Incr(ctx context.Context, key string) (int64, error) {
	return s.connection.Incr(ctx, key)
}

// IncrBy increments a key by a value.
func (s *Service) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return s.connection.IncrBy(ctx, key, value)
}

// Decr decrements a key by 1.
func (s *Service) Decr(ctx context.Context, key string) (int64, error) {
	return s.connection.Decr(ctx, key)
}

// DecrBy decrements a key by a value.
func (s *Service) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return s.connection.DecrBy(ctx, key, value)
}

// Open creates and connects a new KV service using the specified adapter.
func Open(ctx context.Context, adapter adapter.Adapter, config *adapter.Config) (*Service, error) {
	// Create service
	service := NewService(adapter, config)

	// Connect
	if err := service.Connect(ctx); err != nil {
		return nil, err
	}

	return service, nil
}

// OpenWithName creates and connects a new KV service using the specified adapter name.
func OpenWithName(ctx context.Context, adapterName string, config *adapter.Config, opts ...adapter.Option) (*Service, error) {
	// Apply options to config
	for _, opt := range opts {
		opt(config)
	}

	// Get adapter from registry
	adpt, err := adapter.Get(adapterName)
	if err != nil {
		return nil, store.WrapDriverError(err, adapterName, "get adapter")
	}

	return Open(ctx, adpt, config)
}
