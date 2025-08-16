package adapter

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgreSQLAdapter implements the Adapter interface for PostgreSQL.
type PostgreSQLAdapter struct {
	db *sql.DB
}

// NewPostgreSQLAdapter creates a new PostgreSQL adapter.
func NewPostgreSQLAdapter() *PostgreSQLAdapter {
	return &PostgreSQLAdapter{}
}

// Name returns the adapter name.
func (a *PostgreSQLAdapter) Name() string {
	return "postgresql"
}

// Connect establishes a connection to PostgreSQL.
func (a *PostgreSQLAdapter) Connect(ctx context.Context, config *Config) (*sql.DB, error) {
	connStr := a.ConnectionString(config)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL connection: %w", err)
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
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	a.db = db
	return db, nil
}

// ConnectionString constructs a PostgreSQL connection string.
func (a *PostgreSQLAdapter) ConnectionString(config *Config) string {
	var parts []string

	if config.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", config.Host))
	}
	if config.Port > 0 {
		parts = append(parts, fmt.Sprintf("port=%d", config.Port))
	}
	if config.DBName != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", config.DBName))
	}
	if config.User != "" {
		parts = append(parts, fmt.Sprintf("user=%s", config.User))
	}
	if config.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", config.Password))
	}

	// SSL mode
	sslMode := config.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	parts = append(parts, fmt.Sprintf("sslmode=%s", sslMode))

	// Add additional connection parameters
	for key, value := range config.Options {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}

	return strings.Join(parts, " ")
}

// SupportsMigrations indicates PostgreSQL supports migrations.
func (a *PostgreSQLAdapter) SupportsMigrations() bool {
	return true
}

// MigrationTableName returns the migration table name.
func (a *PostgreSQLAdapter) MigrationTableName() string {
	return "schema_migrations"
}

// MigrationTableSQL returns the SQL to create the migration table.
func (a *PostgreSQLAdapter) MigrationTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`
}

// SupportsTransactions indicates PostgreSQL supports transactions.
func (a *PostgreSQLAdapter) SupportsTransactions() bool {
	return true
}

// DefaultTxOptions returns default transaction options for PostgreSQL.
func (a *PostgreSQLAdapter) DefaultTxOptions() *sql.TxOptions {
	return &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	}
}

// SupportsUUID indicates PostgreSQL supports UUIDs.
func (a *PostgreSQLAdapter) SupportsUUID() bool {
	return true
}

// SupportsJSON indicates PostgreSQL supports JSON.
func (a *PostgreSQLAdapter) SupportsJSON() bool {
	return true
}

// SupportsFullTextSearch indicates PostgreSQL supports full-text search.
func (a *PostgreSQLAdapter) SupportsFullTextSearch() bool {
	return true
}

// IsUniqueConstraintViolation checks if an error is a unique constraint violation.
func (a *PostgreSQLAdapter) IsUniqueConstraintViolation(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "unique constraint") ||
		strings.Contains(errStr, "duplicate key")
}

// IsForeignKeyViolation checks if an error is a foreign key violation.
func (a *PostgreSQLAdapter) IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "foreign key") ||
		strings.Contains(errStr, "violates foreign key constraint")
}

// IsConnectionError checks if an error is a connection-related error.
func (a *PostgreSQLAdapter) IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "connect") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection reset")
}

// Close releases resources held by the adapter.
func (a *PostgreSQLAdapter) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}
