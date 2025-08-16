package adapter

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

// MySQLAdapter implements the Adapter interface for MySQL.
type MySQLAdapter struct {
	db *sql.DB
}

// NewMySQLAdapter creates a new MySQL adapter.
func NewMySQLAdapter() *MySQLAdapter {
	return &MySQLAdapter{}
}

// Name returns the adapter name.
func (a *MySQLAdapter) Name() string {
	return "mysql"
}

// Connect establishes a connection to MySQL.
func (a *MySQLAdapter) Connect(ctx context.Context, config *Config) (*sql.DB, error) {
	connStr := a.ConnectionString(config)

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	// Configure connection pool
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	a.db = db
	return db, nil
}

// ConnectionString constructs a MySQL connection string.
// Format: [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
func (a *MySQLAdapter) ConnectionString(config *Config) string {
	var connStr strings.Builder

	// User credentials
	if config.User != "" {
		connStr.WriteString(config.User)
		if config.Password != "" {
			connStr.WriteString(":")
			connStr.WriteString(config.Password)
		}
		connStr.WriteString("@")
	}

	// Protocol and address
	if config.Host != "" || config.Port > 0 {
		connStr.WriteString("tcp(")
		if config.Host != "" {
			connStr.WriteString(config.Host)
		} else {
			connStr.WriteString("localhost")
		}
		if config.Port > 0 {
			connStr.WriteString(fmt.Sprintf(":%d", config.Port))
		}
		connStr.WriteString(")")
	}

	// Database name
	connStr.WriteString("/")
	if config.DBName != "" {
		connStr.WriteString(config.DBName)
	}

	// Parameters
	var params []string

	// SSL mode
	if config.SSLMode != "" {
		params = append(params, fmt.Sprintf("tls=%s", config.SSLMode))
	} else {
		params = append(params, "tls=false") // Default to no SSL
	}

	// Default parameters for better compatibility
	params = append(params, "parseTime=true")
	params = append(params, "charset=utf8mb4")
	params = append(params, "collation=utf8mb4_unicode_ci")

	// Add additional options
	for key, value := range config.Options {
		params = append(params, fmt.Sprintf("%s=%s", key, value))
	}

	if len(params) > 0 {
		connStr.WriteString("?")
		connStr.WriteString(strings.Join(params, "&"))
	}

	return connStr.String()
}

// SupportsMigrations indicates MySQL supports migrations.
func (a *MySQLAdapter) SupportsMigrations() bool {
	return true
}

// MigrationTableName returns the migration table name.
func (a *MySQLAdapter) MigrationTableName() string {
	return "schema_migrations"
}

// MigrationTableSQL returns the SQL to create the migration table.
func (a *MySQLAdapter) MigrationTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
}

// SupportsTransactions indicates MySQL supports transactions.
func (a *MySQLAdapter) SupportsTransactions() bool {
	return true
}

// DefaultTxOptions returns default transaction options for MySQL.
func (a *MySQLAdapter) DefaultTxOptions() *sql.TxOptions {
	return &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead, // MySQL default
		ReadOnly:  false,
	}
}

// SupportsUUID indicates MySQL has limited UUID support.
func (a *MySQLAdapter) SupportsUUID() bool {
	return false // No native UUID type, but can store as CHAR(36) or BINARY(16)
}

// SupportsJSON indicates MySQL supports JSON (since version 5.7).
func (a *MySQLAdapter) SupportsJSON() bool {
	return true
}

// SupportsFullTextSearch indicates MySQL supports full-text search.
func (a *MySQLAdapter) SupportsFullTextSearch() bool {
	return true
}

// IsUniqueConstraintViolation checks if an error is a unique constraint violation.
func (a *MySQLAdapter) IsUniqueConstraintViolation(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "duplicate entry") ||
		strings.Contains(errStr, "unique constraint") ||
		strings.Contains(errStr, "error 1062")
}

// IsForeignKeyViolation checks if an error is a foreign key violation.
func (a *MySQLAdapter) IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "foreign key constraint") ||
		strings.Contains(errStr, "cannot add or update a child row") ||
		strings.Contains(errStr, "cannot delete or update a parent row") ||
		strings.Contains(errStr, "error 1452") ||
		strings.Contains(errStr, "error 1451")
}

// IsConnectionError checks if an error is a connection-related error.
func (a *MySQLAdapter) IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "connect") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "server has gone away") ||
		strings.Contains(errStr, "error 2003") ||
		strings.Contains(errStr, "error 2006")
}

// Close releases resources held by the adapter.
func (a *MySQLAdapter) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}
