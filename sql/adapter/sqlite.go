package adapter

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"store"
	"strings"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// SQLiteAdapter implements the Adapter interface for SQLite.
type SQLiteAdapter struct {
	*BaseSQLAdapter
}

// NewSQLiteAdapter creates a new SQLite adapter.
func NewSQLiteAdapter() *SQLiteAdapter {
	return &SQLiteAdapter{
		BaseSQLAdapter: NewBaseSQLAdapter("sqlite3", "sqlite"),
	}
}

// Connect establishes a connection to SQLite.
func (a *SQLiteAdapter) Connect(ctx context.Context, config *store.Config) (*sql.DB, error) {
	connStr := a.ConnectionString(config)

	// SQLite-specific connection handling
	db, err := a.BaseSQLAdapter.Connect(ctx, config, connStr)
	if err != nil {
		return nil, err
	}

	// SQLite-specific optimizations
	a.configureSQLiteOptimizations(db)

	return db, nil
}

// configureSQLiteOptimizations applies SQLite-specific performance settings.
func (a *SQLiteAdapter) configureSQLiteOptimizations(db *sql.DB) {
	// Enable WAL mode for better concurrency
	db.Exec("PRAGMA journal_mode=WAL")

	// Set synchronous to NORMAL for better performance
	db.Exec("PRAGMA synchronous=NORMAL")

	// Enable foreign keys
	db.Exec("PRAGMA foreign_keys=ON")

	// Set cache size (negative value = KB)
	db.Exec("PRAGMA cache_size=-64000") // 64MB cache
}

// ConnectionString constructs a SQLite connection string.
func (a *SQLiteAdapter) ConnectionString(config *store.Config) string {
	// For SQLite, use FilePath or Database field as the file path
	dbPath := config.FilePath
	if dbPath == "" {
		dbPath = config.Database
	}
	if dbPath == "" {
		dbPath = ":memory:" // In-memory database
	}

	// Expand relative paths
	if dbPath != ":memory:" && !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(".", dbPath)
	}

	// Add query parameters if provided
	var params []string
	for key, value := range config.Options {
		params = append(params, fmt.Sprintf("%s=%s", key, value))
	}

	if len(params) > 0 {
		return fmt.Sprintf("%s?%s", dbPath, strings.Join(params, "&"))
	}

	return dbPath
}

// SQLite-specific overrides

// MigrationTableSQL returns SQLite-specific migration table SQL.
func (a *SQLiteAdapter) MigrationTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`
}

// DefaultTxOptions returns SQLite-specific transaction options.
func (a *SQLiteAdapter) DefaultTxOptions() *sql.TxOptions {
	return &sql.TxOptions{
		Isolation: sql.LevelSerializable, // SQLite uses serializable by default
		ReadOnly:  false,
	}
}

// SQLite-specific error detection
func (a *SQLiteAdapter) IsKeyNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// SQLite: no rows in result set
	return strings.Contains(err.Error(), "no rows in result set")
}

// SQLite-specific capability methods

// SupportsReturning indicates SQLite supports RETURNING clause (since version 3.35.0).
func (a *SQLiteAdapter) SupportsReturning() bool {
	return true
}

// SupportsUpsert indicates SQLite supports INSERT OR REPLACE / ON CONFLICT.
func (a *SQLiteAdapter) SupportsUpsert() bool {
	return true
}

// QuoteIdentifier quotes a SQLite identifier.
func (a *SQLiteAdapter) QuoteIdentifier(identifier string) string {
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(identifier, `"`, `""`))
}

// GetDialect returns the SQL dialect for SQLite.
func (a *SQLiteAdapter) GetDialect() string {
	return "sqlite"
}

// SQLite-specific methods

// SupportsWAL indicates SQLite supports Write-Ahead Logging.
func (a *SQLiteAdapter) SupportsWAL() bool {
	return true
}

// IsEmbedded indicates SQLite is an embedded database.
func (a *SQLiteAdapter) IsEmbedded() bool {
	return true
}
