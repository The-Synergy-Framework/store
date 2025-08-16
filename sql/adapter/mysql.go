package adapter

import (
	"context"
	"database/sql"
	"fmt"
	"store"
	"strings"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

// MySQLAdapter implements the Adapter interface for MySQL.
type MySQLAdapter struct {
	*BaseSQLAdapter
}

// NewMySQLAdapter creates a new MySQL adapter.
func NewMySQLAdapter() *MySQLAdapter {
	return &MySQLAdapter{
		BaseSQLAdapter: NewBaseSQLAdapter("mysql", "mysql"),
	}
}

// Connect establishes a connection to MySQL.
func (a *MySQLAdapter) Connect(ctx context.Context, config *store.Config) (*sql.DB, error) {
	connStr := a.ConnectionString(config)
	return a.BaseSQLAdapter.Connect(ctx, config, connStr)
}

// ConnectionString constructs a MySQL connection string.
// Format: [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
func (a *MySQLAdapter) ConnectionString(config *store.Config) string {
	var connStr strings.Builder

	// User credentials
	if config.Username != "" {
		connStr.WriteString(config.Username)
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
	if config.Database != "" {
		connStr.WriteString(config.Database)
	}

	// Parameters
	var params []string

	// Always add parseTime for proper time handling
	params = append(params, "parseTime=true")

	// Add charset if not specified
	hasCharset := false
	for key := range config.Options {
		if strings.ToLower(key) == "charset" {
			hasCharset = true
			break
		}
	}
	if !hasCharset {
		params = append(params, "charset=utf8mb4")
	}

	// Add custom options
	for key, value := range config.Options {
		params = append(params, fmt.Sprintf("%s=%s", key, value))
	}

	if len(params) > 0 {
		connStr.WriteString("?")
		connStr.WriteString(strings.Join(params, "&"))
	}

	return connStr.String()
}

// MySQL-specific overrides

// MigrationTableSQL returns MySQL-specific migration table SQL.
func (a *MySQLAdapter) MigrationTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
}

// DefaultTxOptions returns MySQL-specific transaction options.
func (a *MySQLAdapter) DefaultTxOptions() *sql.TxOptions {
	return &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead, // MySQL default
		ReadOnly:  false,
	}
}

// MySQL-specific error detection
func (a *MySQLAdapter) IsKeyNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// MySQL: no rows in result set
	return strings.Contains(err.Error(), "no rows in result set")
}

// MySQL-specific capability methods

// SupportsReturning indicates MySQL does NOT support RETURNING clause (before MySQL 8.0.21).
func (a *MySQLAdapter) SupportsReturning() bool {
	return false
}

// SupportsUpsert indicates MySQL supports ON DUPLICATE KEY UPDATE.
func (a *MySQLAdapter) SupportsUpsert() bool {
	return true
}

// QuoteIdentifier quotes a MySQL identifier.
func (a *MySQLAdapter) QuoteIdentifier(identifier string) string {
	return fmt.Sprintf("`%s`", strings.ReplaceAll(identifier, "`", "``"))
}

// GetDialect returns the SQL dialect for MySQL.
func (a *MySQLAdapter) GetDialect() string {
	return "mysql"
}
