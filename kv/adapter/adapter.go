package adapter

import (
	"context"
	"time"

	"store"
)

// Adapter represents a key-value store adapter (Redis, Memory, etc.).
// This follows the guard adapter pattern for pluggable backends.
type Adapter interface {
	// Name returns the adapter's unique identifier.
	Name() string

	// Connect establishes a connection to the key-value store.
	Connect(ctx context.Context, config *Config) (Connection, error)

	// ConnectionString builds the connection string from config.
	ConnectionString(config *Config) string

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

// Config holds KV adapter configuration.
// It extends the shared base config with KV-specific fields.
type Config struct {
	store.BaseConfig

	// KV-specific fields
	Database int // Redis database number

	// KV-specific pooling
	MaxActiveConns int // KV stores may use different pooling concepts

	// KV-specific timeouts
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// Security
	TLS     bool
	TLSCert string
	TLSKey  string
	TLSCA   string
}

// Option configures a KV adapter.
type Option func(*Config)

// WithMetrics enables metrics collection.
func WithMetrics(enabled bool, labels map[string]string) Option {
	return func(c *Config) {
		c.EnableMetrics = enabled
		c.MetricLabels = labels
	}
}

// WithPooling configures connection pooling.
func WithPooling(maxIdle, maxActive int, maxLifetime, maxIdleTime time.Duration) Option {
	return func(c *Config) {
		c.MaxIdleConns = maxIdle
		c.MaxActiveConns = maxActive
		c.ConnMaxLifetime = maxLifetime
		c.ConnMaxIdleTime = maxIdleTime
	}
}

// WithTimeouts configures operation timeouts.
func WithTimeouts(connect, read, write time.Duration) Option {
	return func(c *Config) {
		c.ConnectTimeout = connect
		c.ReadTimeout = read
		c.WriteTimeout = write
	}
}

// WithConnection configures basic connection parameters.
func WithConnection(host string, port int, username, password string) Option {
	return func(c *Config) {
		c.Host = host
		c.Port = port
		c.Username = username
		c.Password = password
	}
}

// WithDatabase configures database selection (for Redis).
func WithDatabase(database int) Option {
	return func(c *Config) {
		c.Database = database
	}
}

// WithTLS configures TLS settings.
func WithTLS(enabled bool, cert, key, ca string) Option {
	return func(c *Config) {
		c.TLS = enabled
		c.TLSCert = cert
		c.TLSKey = key
		c.TLSCA = ca
	}
}

// DefaultConfig returns a KV configuration with sensible defaults.
func DefaultConfig() Config {
	baseConfig := store.DefaultConfig()
	return Config{
		BaseConfig:     baseConfig,
		Database:       0, // Redis default database
		MaxActiveConns: 25,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		TLS:            false,
	}
}
