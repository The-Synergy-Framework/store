package adapter

import (
	"context"
	"database/sql"
	"store"
)

// BaseSQLAdapter provides common functionality for all SQL adapters.
type BaseSQLAdapter struct {
	db         *sql.DB
	driverName string
	name       AdapterName
}

// NewBaseSQLAdapter creates a new base SQL adapter.
func NewBaseSQLAdapter(driverName string, name AdapterName) *BaseSQLAdapter {
	return &BaseSQLAdapter{
		driverName: driverName,
		name:       name,
	}
}

// Name returns the adapter name.
func (a *BaseSQLAdapter) Name() AdapterName {
	return a.name
}

// Connect establishes a database connection with common configuration.
// This eliminates ~50 lines of identical code across all SQL adapters.
func (a *BaseSQLAdapter) Connect(ctx context.Context, config *store.Config, connectionString string) (*sql.DB, error) {
	// Open database connection
	db, err := sql.Open(a.driverName, connectionString)
	if err != nil {
		return nil, store.WrapConnectionError(
			err, "connect", a.driverName, config.Host)
	}

	// Configure connection pool - identical across all SQL adapters
	a.configureConnectionPool(db, config)

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, store.WrapConnectionError(
			err, "ping", a.driverName, config.Host)
	}

	a.db = db
	return db, nil
}

// configureConnectionPool sets up connection pooling - identical across all adapters.
func (a *BaseSQLAdapter) configureConnectionPool(db *sql.DB, config *store.Config) {
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
}

// Close closes the database connection.
func (a *BaseSQLAdapter) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// DB returns the underlying database connection.
func (a *BaseSQLAdapter) DB() *sql.DB {
	return a.db
}

// Common capability methods - identical across all SQL adapters
func (a *BaseSQLAdapter) SupportsMigrations() bool {
	return true
}

func (a *BaseSQLAdapter) SupportsTransactions() bool {
	return true
}

func (a *BaseSQLAdapter) MigrationTableName() string {
	return "schema_migrations"
}

// GetMigrationTableSQL returns database-specific migration table SQL.
// Each adapter can override this for database-specific syntax.
func (a *BaseSQLAdapter) GetMigrationTableSQL() string {
	// Default implementation - adapters can override
	return `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
}

// GetDefaultTxOptions returns default transaction options.
// Each adapter can override this for database-specific defaults.
func (a *BaseSQLAdapter) GetDefaultTxOptions() *sql.TxOptions {
	// Default implementation - adapters can override
	return &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	}
}

// Common error checking methods - similar patterns across adapters
func (a *BaseSQLAdapter) IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Common connection error patterns across databases
	connectionErrors := []string{
		"connection refused",
		"connection reset",
		"connection closed",
		"network is unreachable",
		"timeout",
		"driver: bad connection",
	}

	for _, pattern := range connectionErrors {
		if contains(errStr, pattern) {
			return true
		}
	}
	return false
}

func (a *BaseSQLAdapter) IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	timeoutErrors := []string{
		"timeout",
		"context deadline exceeded",
		"context canceled",
	}

	for _, pattern := range timeoutErrors {
		if contains(errStr, pattern) {
			return true
		}
	}
	return false
}

func (a *BaseSQLAdapter) SupportsUUID() bool {
	return false
}

func (a *BaseSQLAdapter) SupportsJSON() bool {
	return false
}

func (a *BaseSQLAdapter) SupportsFullTextSearch() bool {
	return false
}

func (a *BaseSQLAdapter) IsUniqueConstraintViolation(err error) bool {
	if err == nil {
		return false
	}
	errStr := toLower(err.Error())
	uniqueErrors := []string{
		"unique constraint",
		"duplicate key",
		"duplicate entry",
	}

	for _, pattern := range uniqueErrors {
		if contains(errStr, pattern) {
			return true
		}
	}
	return false
}

func (a *BaseSQLAdapter) IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	errStr := toLower(err.Error())
	fkErrors := []string{
		"foreign key constraint",
		"foreign key",
		"violates foreign key",
	}

	for _, pattern := range fkErrors {
		if contains(errStr, pattern) {
			return true
		}
	}
	return false
}

func (a *BaseSQLAdapter) IsKeyNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "no rows in result set")
}

// Helper function for case-insensitive string contains
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexOfIgnoreCase(s, substr) >= 0))
}

func indexOfIgnoreCase(s, substr string) int {
	sLower := toLower(s)
	substrLower := toLower(substr)

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' {
			result[i] = b + 32
		} else {
			result[i] = b
		}
	}
	return string(result)
}
