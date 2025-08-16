package adapter

import (
	"context"
	"database/sql"
	"store"
)

type AdapterName string

// Adapter represents a SQL database adapter (PostgreSQL, MySQL, SQLite).
// This follows the guard adapter pattern for pluggable backends.
type Adapter interface {
	// Name returns the adapter's unique identifier.
	Name() AdapterName

	// Connect establishes a connection to the database.
	Connect(ctx context.Context, config *store.Config) (*sql.DB, error)

	// ConnectionString builds the connection string from config.
	ConnectionString(config *store.Config) string

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
