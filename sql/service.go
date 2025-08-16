package sqlstore

import (
	"context"
	"database/sql"
	"time"

	"core/entity"
	"store"
	"store/sql/adapter"
)

// Service wraps a SQL adapter and provides the database service interface.
// This follows the guard service pattern.
type Service struct {
	adapter adapter.Adapter
	db      *sql.DB
	config  *adapter.Config
}

// Ensure Service implements the service interface.
var _ store.Service = (*Service)(nil)

// NewService creates a new SQL service with the given adapter.
func NewService(adpt adapter.Adapter, config *adapter.Config) *Service {
	return &Service{
		adapter: adpt,
		config:  config,
	}
}

// Connect establishes the database connection.
func (s *Service) Connect(ctx context.Context) error {
	db, err := s.adapter.Connect(ctx, s.config)
	if err != nil {
		return store.WrapConnectionError(err, "connect", s.adapter.Name(), s.config.Host)
	}

	// Configure connection pool
	if s.config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(s.config.MaxOpenConns)
	}
	if s.config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(s.config.MaxIdleConns)
	}
	if s.config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(s.config.ConnMaxLifetime)
	}
	if s.config.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(s.config.ConnMaxIdleTime)
	}

	// Test connection
	pingCtx := ctx
	var cancel context.CancelFunc
	if s.config.ConnectTimeout > 0 {
		pingCtx, cancel = context.WithTimeout(ctx, s.config.ConnectTimeout)
		defer cancel()
	}

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return store.WrapConnectionError(err, "ping", s.adapter.Name(), s.config.Host)
	}

	s.db = db
	return nil
}

// DB returns the underlying database connection.
func (s *Service) DB() *sql.DB {
	return s.db
}

// Adapter returns the underlying adapter.
func (s *Service) Adapter() adapter.Adapter {
	return s.adapter
}

// Close closes the database connection.
func (s *Service) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Stats returns database connection statistics.
func (s *Service) Stats() interface{} {
	if s.db != nil {
		return s.db.Stats()
	}
	return sql.DBStats{}
}

// NewRepository creates a new repository for the given entity type.
func (s *Service) NewRepository(entity entity.Entity) store.Repository {
	return NewRepository(s, entity)
}

// Repository creates a new repository for the given entity type (alias for NewRepository).
func (s *Service) Repository(entity entity.Entity) *Repository {
	return NewRepository(s, entity)
}

// WithTimeout creates a context with timeout for operations.
func (s *Service) WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// QueryExecutor returns a new query executor.
func (s *Service) QueryExecutor() *QueryExecutor {
	return NewQueryExecutor(s.db)
}

// TransactionHandler returns a new transaction handler.
func (s *Service) TransactionHandler() *TransactionHandler {
	return NewTransactionHandler(s.db, s.Adapter())
}

// ExecuteSQL executes raw SQL (for migrations, table creation, etc.).
func (s *Service) ExecuteSQL(ctx context.Context, query string, args ...interface{}) error {
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return store.WrapQueryError(err, "execute_sql", "", query, args)
	}
	return nil
}

// Open creates and connects a new SQL service using the specified adapter.
func Open(ctx context.Context, adapter adapter.Adapter, config *adapter.Config) (*Service, error) {
	// Create service
	service := NewService(adapter, config)

	// Connect
	if err := service.Connect(ctx); err != nil {
		return nil, err
	}

	return service, nil
}

// OpenWithName creates and connects a new SQL service using the specified adapter name.
func OpenWithName(ctx context.Context, adapterName string, config *adapter.Config, opts ...adapter.Option) (*Service, error) {
	// Apply options to config
	for _, opt := range opts {
		opt(config)
	}

	// Get adapter from registry
	adpt, err := adapter.Get(adapterName)
	if err != nil {
		return nil, store.WrapDriverError(err, adapterName, "get adapter")
	}

	return Open(ctx, adpt, config)
}
