package store

import (
	"fmt"
	"time"
)

// Config holds configuration for any storage backend.
// This unified config works for SQL, KV, and file storage.
type Config struct {
	// Backend type
	Type string `json:"type"` // "postgres", "mysql", "sqlite", "redis", "memory", "filesystem"

	// Connection details (used by SQL and network-based backends)
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"` // database name for SQL, bucket/namespace for others
	Username string `json:"username"`
	Password string `json:"password"`

	// File storage specific
	FilePath string `json:"file_path"` // for SQLite file path or filesystem root

	// Connection pooling
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`

	// Timeouts
	ConnectTimeout time.Duration `json:"connect_timeout"`
	QueryTimeout   time.Duration `json:"query_timeout"`

	// SSL/Security
	SSLMode string `json:"ssl_mode"` // "disable", "require", "verify-full"

	// Performance
	EnableMetrics bool `json:"enable_metrics"`

	// Backend-specific options (escape hatch for special settings)
	Options map[string]string `json:"options"`
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Type:            "",
		Host:            "localhost",
		Port:            0, // Will be set by adapter defaults
		Database:        "",
		Username:        "",
		Password:        "",
		FilePath:        "",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 1 * time.Hour,
		ConnectTimeout:  30 * time.Second,
		QueryTimeout:    30 * time.Second,
		SSLMode:         "disable",
		EnableMetrics:   false,
		Options:         make(map[string]string),
	}
}

// PostgreSQLConfig returns a config with PostgreSQL defaults.
func PostgreSQLConfig(database, username, password string) Config {
	config := DefaultConfig()
	config.Type = "postgres"
	config.Port = 5432
	config.Database = database
	config.Username = username
	config.Password = password
	config.SSLMode = "disable"
	return config
}

// MySQLConfig returns a config with MySQL defaults.
func MySQLConfig(database, username, password string) Config {
	config := DefaultConfig()
	config.Type = "mysql"
	config.Port = 3306
	config.Database = database
	config.Username = username
	config.Password = password
	return config
}

// SQLiteConfig returns a config for SQLite.
func SQLiteConfig(filePath string) Config {
	config := DefaultConfig()
	config.Type = "sqlite"
	config.FilePath = filePath
	config.MaxOpenConns = 1 // SQLite doesn't support multiple connections well
	return config
}

// MemoryConfig returns a config for in-memory storage.
func MemoryConfig() Config {
	config := DefaultConfig()
	config.Type = "memory"
	return config
}

// Validate performs basic validation on the config.
func (c *Config) Validate() error {
	if c.Type == "" {
		return NewConfigError("type cannot be empty")
	}

	switch c.Type {
	case "postgres", "mysql":
		if c.Database == "" {
			return NewConfigError("database name required for " + c.Type)
		}
		if c.Username == "" {
			return NewConfigError("username required for " + c.Type)
		}
	case "sqlite":
		if c.FilePath == "" {
			return NewConfigError("file path required for SQLite")
		}
	case "memory":
		// No validation needed for memory
	default:
		return NewConfigError("unsupported type: " + c.Type)
	}

	return nil
}

// ConnectionString builds a connection string for the backend.
func (c *Config) ConnectionString() string {
	switch c.Type {
	case "postgres":
		return c.postgresConnectionString()
	case "mysql":
		return c.mysqlConnectionString()
	case "sqlite":
		return c.FilePath
	default:
		return ""
	}
}

func (c *Config) postgresConnectionString() string {
	host := c.Host
	if c.Port > 0 {
		host += fmt.Sprintf(":%d", c.Port)
	}

	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		c.Username, c.Password, host, c.Database, c.SSLMode)
}

func (c *Config) mysqlConnectionString() string {
	host := c.Host
	if c.Port > 0 {
		host += fmt.Sprintf(":%d", c.Port)
	}

	return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true",
		c.Username, c.Password, host, c.Database)
}

// Legacy support - remove these after migration
type BaseConfig = Config // Alias for backward compatibility
