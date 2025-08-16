package adapter

import (
	"context"
	"database/sql"
	"fmt"
	"store"
	"strings"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgreSQLAdapter implements the Adapter interface for PostgreSQL.
type PostgreSQLAdapter struct {
	*BaseSQLAdapter
}

// NewPostgreSQLAdapter creates a new PostgreSQL adapter.
func NewPostgreSQLAdapter() *PostgreSQLAdapter {
	return &PostgreSQLAdapter{
		BaseSQLAdapter: NewBaseSQLAdapter("postgres", "postgresql"),
	}
}

// Connect establishes a connection to PostgreSQL.
func (a *PostgreSQLAdapter) Connect(ctx context.Context, config *store.Config) (*sql.DB, error) {
	connStr := a.ConnectionString(config)
	return a.BaseSQLAdapter.Connect(ctx, config, connStr)
}

// ConnectionString constructs a PostgreSQL connection string.
func (a *PostgreSQLAdapter) ConnectionString(config *store.Config) string {
	var parts []string

	if config.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", config.Host))
	}
	if config.Port > 0 {
		parts = append(parts, fmt.Sprintf("port=%d", config.Port))
	}
	if config.Database != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", config.Database))
	}
	if config.Username != "" {
		parts = append(parts, fmt.Sprintf("user=%s", config.Username))
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

// PostgreSQL-specific overrides

// MigrationTableSQL returns PostgreSQL-specific migration table SQL.
func (a *PostgreSQLAdapter) MigrationTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`
}

// DefaultTxOptions returns PostgreSQL-specific transaction options.
func (a *PostgreSQLAdapter) DefaultTxOptions() *sql.TxOptions {
	return &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	}
}

// PostgreSQL-specific error detection
func (a *PostgreSQLAdapter) IsKeyNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL: no rows in result set
	return strings.Contains(err.Error(), "no rows in result set")
}

// PostgreSQL-specific capability methods (if different from base)
// Note: Most capabilities are inherited from BaseSQLAdapter

// SupportsReturning indicates PostgreSQL supports RETURNING clause.
func (a *PostgreSQLAdapter) SupportsReturning() bool {
	return true
}

// SupportsUpsert indicates PostgreSQL supports ON CONFLICT (UPSERT).
func (a *PostgreSQLAdapter) SupportsUpsert() bool {
	return true
}

// QuoteIdentifier quotes a PostgreSQL identifier.
func (a *PostgreSQLAdapter) QuoteIdentifier(identifier string) string {
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(identifier, `"`, `""`))
}

// GetDialect returns the SQL dialect for PostgreSQL.
func (a *PostgreSQLAdapter) GetDialect() string {
	return "postgresql"
}
