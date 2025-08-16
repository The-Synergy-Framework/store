package store

import (
	"time"
)

// Option configures a store configuration.
type Option func(*Config)

// Database connection options

// WithConnection sets basic connection parameters for network-based backends.
func WithConnection(host string, port int, username, password, database string) Option {
	return func(c *Config) {
		c.Host = host
		c.Port = port
		c.Username = username
		c.Password = password
		c.Database = database
	}
}

// WithHost sets the connection host.
func WithHost(host string) Option {
	return func(c *Config) {
		c.Host = host
	}
}

// WithPort sets the connection port.
func WithPort(port int) Option {
	return func(c *Config) {
		c.Port = port
	}
}

// WithCredentials sets username and password.
func WithCredentials(username, password string) Option {
	return func(c *Config) {
		c.Username = username
		c.Password = password
	}
}

// WithDatabase sets the database name.
func WithDatabase(database string) Option {
	return func(c *Config) {
		c.Database = database
	}
}

// WithFilePath sets the file path for file-based backends (SQLite, filesystem).
func WithFilePath(path string) Option {
	return func(c *Config) {
		c.FilePath = path
	}
}

// WithPooling configures connection pooling settings.
func WithPooling(maxOpen, maxIdle int, maxLifetime time.Duration) Option {
	return func(c *Config) {
		c.MaxOpenConns = maxOpen
		c.MaxIdleConns = maxIdle
		c.ConnMaxLifetime = maxLifetime
	}
}

// WithMaxOpenConns sets the maximum number of open connections.
func WithMaxOpenConns(max int) Option {
	return func(c *Config) {
		c.MaxOpenConns = max
	}
}

// WithMaxIdleConns sets the maximum number of idle connections.
func WithMaxIdleConns(max int) Option {
	return func(c *Config) {
		c.MaxIdleConns = max
	}
}

// WithConnMaxLifetime sets the maximum connection lifetime.
func WithConnMaxLifetime(lifetime time.Duration) Option {
	return func(c *Config) {
		c.ConnMaxLifetime = lifetime
	}
}

// Timeout options

// WithTimeouts configures operation timeouts.
func WithTimeouts(connect, query time.Duration) Option {
	return func(c *Config) {
		c.ConnectTimeout = connect
		c.QueryTimeout = query
	}
}

// WithConnectTimeout sets the connection timeout.
func WithConnectTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.ConnectTimeout = timeout
	}
}

// WithQueryTimeout sets the query timeout.
func WithQueryTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.QueryTimeout = timeout
	}
}

// Security options

// WithSSL configures SSL/TLS settings.
func WithSSL(mode string) Option {
	return func(c *Config) {
		c.SSLMode = mode
	}
}

// WithSSLDisabled disables SSL (equivalent to WithSSL("disable")).
func WithSSLDisabled() Option {
	return func(c *Config) {
		c.SSLMode = "disable"
	}
}

// WithSSLRequired requires SSL (equivalent to WithSSL("require")).
func WithSSLRequired() Option {
	return func(c *Config) {
		c.SSLMode = "require"
	}
}

// Observability options

// WithMetrics enables metrics collection.
func WithMetrics(enabled bool) Option {
	return func(c *Config) {
		c.EnableMetrics = enabled
	}
}

// WithMetricsEnabled enables metrics collection.
func WithMetricsEnabled() Option {
	return func(c *Config) {
		c.EnableMetrics = true
	}
}

// Custom options

// WithOption sets a custom option in the Options map.
func WithOption(key, value string) Option {
	return func(c *Config) {
		if c.Options == nil {
			c.Options = make(map[string]string)
		}
		c.Options[key] = value
	}
}

// WithOptions sets multiple custom options.
func WithOptions(options map[string]string) Option {
	return func(c *Config) {
		if c.Options == nil {
			c.Options = make(map[string]string)
		}
		for k, v := range options {
			c.Options[k] = v
		}
	}
}

// Backend-specific convenience functions

// PostgreSQLOptions returns common PostgreSQL configuration options.
func PostgreSQLOptions(database, username, password string, opts ...Option) []Option {
	base := []Option{
		func(c *Config) { c.Type = "postgres" },
		func(c *Config) { c.Port = 5432 },
		WithDatabase(database),
		WithCredentials(username, password),
		WithSSLDisabled(),
	}
	return append(base, opts...)
}

// MySQLOptions returns common MySQL configuration options.
func MySQLOptions(database, username, password string, opts ...Option) []Option {
	base := []Option{
		func(c *Config) { c.Type = "mysql" },
		func(c *Config) { c.Port = 3306 },
		WithDatabase(database),
		WithCredentials(username, password),
	}
	return append(base, opts...)
}

// SQLiteOptions returns common SQLite configuration options.
func SQLiteOptions(filePath string, opts ...Option) []Option {
	base := []Option{
		func(c *Config) { c.Type = "sqlite" },
		WithFilePath(filePath),
		WithMaxOpenConns(1), // SQLite works best with single connection
	}
	return append(base, opts...)
}

// MemoryOptions returns common in-memory storage configuration options.
func MemoryOptions(opts ...Option) []Option {
	base := []Option{
		func(c *Config) { c.Type = "memory" },
	}
	return append(base, opts...)
}

// NewConfig creates a new configuration with the given options.
func NewConfig(opts ...Option) Config {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(&config)
	}
	return config
}

// Apply applies multiple options to an existing configuration.
func (c *Config) Apply(opts ...Option) *Config {
	for _, opt := range opts {
		opt(c)
	}
	return c
}
