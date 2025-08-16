package adapter

import (
	"context"
	"database/sql"
	"store"
	"time"
)

// Adapter represents a SQL database adapter (PostgreSQL, MySQL, SQLite).
// This follows the guard adapter pattern for pluggable backends.
type Adapter interface {
	// Name returns the adapter's unique identifier.
	Name() string

	// Connect establishes a connection to the database.
	Connect(ctx context.Context, config *Config) (*sql.DB, error)

	// ConnectionString builds the connection string from config.
	ConnectionString(config *Config) string

	// Database capabilities
	SupportsMigrations() bool
	MigrationTableName() string
	MigrationTableSQL() string
	SupportsTransactions() bool
	DefaultTxOptions() *sql.TxOptions
	SupportsUUID() bool
	SupportsJSON() bool
	SupportsFullTextSearch() bool

	// Error classification
	IsUniqueConstraintViolation(err error) bool
	IsForeignKeyViolation(err error) bool
	IsConnectionError(err error) bool

	// Close releases any resources held by the adapter.
	Close() error
}

// Config holds SQL adapter configuration.
// It extends the shared base config with SQL-specific fields.
type Config struct {
	store.BaseConfig

	// SQL-specific fields
	User    string // some SQL DBs use User instead of Username
	DBName  string // database name
	SSLMode string // SSL configuration

	// SQL-specific pooling
	MaxOpenConns int // SQL databases need max open connections

	// SQL-specific timeouts
	QueryTimeout time.Duration
	TxTimeout    time.Duration
}

// Option configures a SQL adapter.
type Option func(*Config)

// WithMetrics enables metrics collection.
func WithMetrics(enabled bool, labels map[string]string) Option {
	return func(c *Config) {
		c.EnableMetrics = enabled
		c.MetricLabels = labels
	}
}

// WithPooling configures connection pooling.
func WithPooling(maxOpen, maxIdle int, maxLifetime, maxIdleTime time.Duration) Option {
	return func(c *Config) {
		c.MaxOpenConns = maxOpen
		c.MaxIdleConns = maxIdle
		c.ConnMaxLifetime = maxLifetime
		c.ConnMaxIdleTime = maxIdleTime
	}
}

// WithTimeouts configures operation timeouts.
func WithTimeouts(connect, query, tx time.Duration) Option {
	return func(c *Config) {
		c.ConnectTimeout = connect
		c.QueryTimeout = query
		c.TxTimeout = tx
	}
}

// WithDatabase configures database connection.
func WithDatabase(host string, port int, user, password, dbname string) Option {
	return func(c *Config) {
		c.Host = host
		c.Port = port
		c.User = user
		c.Username = user // Set both for compatibility
		c.Password = password
		c.DBName = dbname
	}
}

// WithSSL configures SSL settings.
func WithSSL(sslMode string) Option {
	return func(c *Config) {
		c.SSLMode = sslMode
	}
}

// DefaultConfig returns a SQL configuration with sensible defaults.
func DefaultConfig() Config {
	baseConfig := store.DefaultConfig()
	return Config{
		BaseConfig:   baseConfig,
		User:         "",
		DBName:       "",
		SSLMode:      "disable",
		MaxOpenConns: 25,
		QueryTimeout: 30 * time.Second,
		TxTimeout:    30 * time.Second,
	}
}
