package adapter

import (
	"context"
	"time"

	"store"
)

// Adapter represents a key-value store adapter (Redis, Memory, etc.).
type Adapter interface {
	// Name returns the adapter's unique identifier.
	Name() string

	// Connect establishes a connection to the key-value store.
	Connect(ctx context.Context, config *store.Config) (Connection, error)

	// ConnectionString builds the connection string from config.
	ConnectionString(config *store.Config) string

	// Store capabilities
	SupportsExpiration() bool
	SupportsTransactions() bool
	SupportsPipelining() bool
	SupportsPatternMatching() bool
	SupportsPubSub() bool

	// Data type support
	SupportsLists() bool
	SupportsSets() bool
	SupportsHashes() bool
	SupportsSortedSets() bool
	SupportsStreams() bool

	// Error classification
	IsKeyNotFoundError(err error) bool
	IsConnectionError(err error) bool
	IsTimeoutError(err error) bool

	// Close releases any resources held by the adapter.
	Close() error
}

// Connection represents a connection to a key-value store.
type Connection interface {
	// Basic key-value operations
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Batch operations
	MGet(ctx context.Context, keys []string) (map[string][]byte, error)
	MSet(ctx context.Context, pairs map[string][]byte, expiration time.Duration) error
	MDelete(ctx context.Context, keys []string) error

	// Pattern operations
	Keys(ctx context.Context, pattern string) ([]string, error)
	Scan(ctx context.Context, cursor string, pattern string, count int) ([]string, string, error)

	// Expiration
	Expire(ctx context.Context, key string, expiration time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Atomic operations
	Incr(ctx context.Context, key string) (int64, error)
	IncrBy(ctx context.Context, key string, value int64) (int64, error)
	Decr(ctx context.Context, key string) (int64, error)
	DecrBy(ctx context.Context, key string, value int64) (int64, error)

	// Transaction support (if available)
	Pipeline() Pipeline
	Transaction() Transaction

	// Health and stats
	Ping(ctx context.Context) error
	Stats() interface{}
	Close() error
}

// Pipeline represents a pipeline for batching operations.
type Pipeline interface {
	Get(key string) PipelineCmd
	Set(key string, value []byte, expiration time.Duration) PipelineCmd
	Delete(key string) PipelineCmd
	Exec(ctx context.Context) error
	Discard()
}

// Transaction represents a transaction for atomic operations.
type Transaction interface {
	Get(key string) TransactionCmd
	Set(key string, value []byte, expiration time.Duration) TransactionCmd
	Delete(key string) TransactionCmd
	Watch(keys ...string) error
	Exec(ctx context.Context) error
	Discard()
}

// PipelineCmd represents a command in a pipeline.
type PipelineCmd interface {
	Result() ([]byte, error)
}

// TransactionCmd represents a command in a transaction.
type TransactionCmd interface {
	Result() ([]byte, error)
}

// Config is now just an alias to store.Config - unified configuration!
// KV-specific options (database number, read/write timeouts, TLS settings)
// can be set via store.WithOption() or the Options map.
type Config = store.Config

// Legacy Option type - prefer store.Option in new code
type Option func(*Config)

// WithMetrics has been moved to the root store package.
// Use store.WithMetrics() or store.WithMetricsEnabled() instead.

// Old KV-specific options have been moved to the root store package.
// Use store.WithPooling(), store.WithTimeouts(), store.WithConnection() instead.
// KV-specific timeouts (read/write) can be set via store.WithOption().

// DefaultConfig returns a KV configuration with sensible defaults.
func DefaultConfig() Config {
	config := store.DefaultConfig()
	// Set KV-specific defaults via Options map
	config.Options["database"] = "0" // Redis default database
	config.Options["max_active_conns"] = "25"
	config.Options["read_timeout"] = "30s"
	config.Options["write_timeout"] = "30s"
	config.Options["tls"] = "false"
	return config
}

// Legacy options - use store.WithOption() for KV-specific settings
// Example: store.WithOption("database", "1") for Redis database selection
// Example: store.WithOption("tls", "true") for TLS enablement
