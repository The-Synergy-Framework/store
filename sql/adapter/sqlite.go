package adapter

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// SQLiteAdapter implements the Adapter interface for SQLite.
type SQLiteAdapter struct {
	db *sql.DB
}

// NewSQLiteAdapter creates a new SQLite adapter.
func NewSQLiteAdapter() *SQLiteAdapter {
	return &SQLiteAdapter{}
}

// Name returns the adapter name.
func (a *SQLiteAdapter) Name() string {
	return "sqlite"
}

// Connect establishes a connection to SQLite.
func (a *SQLiteAdapter) Connect(ctx context.Context, config *Config) (*sql.DB, error) {
	connStr := a.ConnectionString(config)

	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite connection: %w", err)
	}

	// Configure connection pool (SQLite specific)
	// SQLite works best with a single connection for writes
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(1) // Default for SQLite
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
		return nil, fmt.Errorf("failed to ping SQLite: %w", err)
	}

	// Enable foreign keys (disabled by default in SQLite)
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	a.db = db
	return db, nil
}

// ConnectionString constructs a SQLite connection string.
func (a *SQLiteAdapter) ConnectionString(config *Config) string {
	// For SQLite, DBName is the file path
	dbPath := config.DBName
	if dbPath == "" {
		dbPath = ":memory:" // In-memory database
	} else {
		// Ensure the path is absolute or relative to current working directory
		if !filepath.IsAbs(dbPath) && !strings.HasPrefix(dbPath, ":") {
			dbPath = filepath.Clean(dbPath)
		}
	}

	// Add SQLite-specific options
	var params []string
	for key, value := range config.Options {
		params = append(params, fmt.Sprintf("%s=%s", key, value))
	}

	if len(params) > 0 {
		return fmt.Sprintf("%s?%s", dbPath, strings.Join(params, "&"))
	}

	return dbPath
}

// SupportsMigrations indicates SQLite supports migrations.
func (a *SQLiteAdapter) SupportsMigrations() bool {
	return true
}

// MigrationTableName returns the migration table name.
func (a *SQLiteAdapter) MigrationTableName() string {
	return "schema_migrations"
}

// MigrationTableSQL returns the SQL to create the migration table.
func (a *SQLiteAdapter) MigrationTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`
}

// SupportsTransactions indicates SQLite supports transactions.
func (a *SQLiteAdapter) SupportsTransactions() bool {
	return true
}

// DefaultTxOptions returns default transaction options for SQLite.
func (a *SQLiteAdapter) DefaultTxOptions() *sql.TxOptions {
	return &sql.TxOptions{
		Isolation: sql.LevelSerializable, // SQLite default
		ReadOnly:  false,
	}
}

// SupportsUUID indicates SQLite does not have native UUID support.
func (a *SQLiteAdapter) SupportsUUID() bool {
	return false // No native UUID type, but can store as TEXT
}

// SupportsJSON indicates SQLite supports JSON (since version 3.38).
func (a *SQLiteAdapter) SupportsJSON() bool {
	return true // JSON1 extension is commonly available
}

// SupportsFullTextSearch indicates SQLite supports FTS.
func (a *SQLiteAdapter) SupportsFullTextSearch() bool {
	return true // FTS5 extension
}

// IsUniqueConstraintViolation checks if an error is a unique constraint violation.
func (a *SQLiteAdapter) IsUniqueConstraintViolation(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "unique constraint") ||
		strings.Contains(errStr, "unique constraint failed")
}

// IsForeignKeyViolation checks if an error is a foreign key violation.
func (a *SQLiteAdapter) IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "foreign key constraint") ||
		strings.Contains(errStr, "foreign key constraint failed")
}

// IsConnectionError checks if an error is a connection-related error.
func (a *SQLiteAdapter) IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "database is locked") ||
		strings.Contains(errStr, "database schema has changed") ||
		strings.Contains(errStr, "no such file") ||
		strings.Contains(errStr, "unable to open database")
}

// Close releases resources held by the adapter.
func (a *SQLiteAdapter) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}
