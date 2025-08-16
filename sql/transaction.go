package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"store"
	"time"

	"store/sql/adapter"
)

type txContextKey struct{}
type txInfoKey struct{}

// TxInfo contains metadata about the current transaction.
type TxInfo struct {
	ReadOnly  bool
	StartTime time.Time
	Options   store.TxOptions
}

// TransactionFromContext extracts an *sql.Tx from context when present.
func TransactionFromContext(ctx context.Context) (*sql.Tx, bool) {
	v := ctx.Value(txContextKey{})
	if v == nil {
		return nil, false
	}
	tx, ok := v.(*sql.Tx)
	return tx, ok
}

// TxInfoFromContext extracts transaction info from context.
func TxInfoFromContext(ctx context.Context) (*TxInfo, bool) {
	v := ctx.Value(txInfoKey{})
	if v == nil {
		return nil, false
	}
	info, ok := v.(*TxInfo)
	return info, ok
}

type TransactionHandler struct {
	db      *sql.DB
	adapter adapter.Adapter
}

func NewTransactionHandler(db *sql.DB, adpt adapter.Adapter) *TransactionHandler {
	return &TransactionHandler{db: db, adapter: adpt}
}

// Ensure TransactionHandler satisfies enhanced interfaces.
var _ store.Transactor = (*TransactionHandler)(nil)
var _ store.TransactionManager = (*TransactionHandler)(nil)

func (t *TransactionHandler) WithTx(ctx context.Context, fn func(context.Context) error) error {
	opts := store.TxOptions{ReadOnly: false}
	return t.WithTxOptions(ctx, opts, fn)
}

func (t *TransactionHandler) WithReadTx(ctx context.Context, fn func(context.Context) error) error {
	opts := store.TxOptions{ReadOnly: true}
	return t.WithTxOptions(ctx, opts, fn)
}

func (t *TransactionHandler) WithTxOptions(ctx context.Context, opts store.TxOptions, fn func(context.Context) error) error {
	// Reuse existing transaction if present
	if existing, ok := TransactionFromContext(ctx); ok && existing != nil {
		return fn(ctx)
	}

	// Apply retry policy if specified
	if opts.RetryPolicy != nil {
		return t.withRetry(ctx, opts, fn)
	}

	return t.executeTx(ctx, opts, fn)
}

func (t *TransactionHandler) HasTx(ctx context.Context) bool {
	_, has := TransactionFromContext(ctx)
	return has
}

func (t *TransactionHandler) IsTxReadOnly(ctx context.Context) bool {
	info, ok := TxInfoFromContext(ctx)
	if !ok {
		return false
	}
	return info.ReadOnly
}

// Advanced transaction management

func (t *TransactionHandler) Savepoint(ctx context.Context, name string) error {
	tx, ok := TransactionFromContext(ctx)
	if !ok {
		return store.NewTransactionError(nil, "savepoint_no_tx")
	}

	query := fmt.Sprintf("SAVEPOINT %s", name)
	_, err := tx.ExecContext(ctx, query)
	if err != nil {
		return store.WrapTransactionError(err, "savepoint")
	}

	return nil
}

func (t *TransactionHandler) RollbackToSavepoint(ctx context.Context, name string) error {
	tx, ok := TransactionFromContext(ctx)
	if !ok {
		return store.NewTransactionError(nil, "rollback_savepoint_no_tx")
	}

	query := fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", name)
	_, err := tx.ExecContext(ctx, query)
	if err != nil {
		return store.WrapTransactionError(err, "rollback_savepoint")
	}

	return nil
}

func (t *TransactionHandler) ReleaseSavepoint(ctx context.Context, name string) error {
	tx, ok := TransactionFromContext(ctx)
	if !ok {
		return store.NewTransactionError(nil, "release_savepoint_no_tx")
	}

	query := fmt.Sprintf("RELEASE SAVEPOINT %s", name)
	_, err := tx.ExecContext(ctx, query)
	if err != nil {
		return store.WrapTransactionError(err, "release_savepoint")
	}

	return nil
}

// Private methods

func (t *TransactionHandler) executeTx(ctx context.Context, opts store.TxOptions, fn func(context.Context) error) error {
	// Apply timeout if specified
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Convert options to SQL transaction options
	sqlOpts := t.toSQLTxOptions(opts)

	tx, err := t.db.BeginTx(ctx, sqlOpts)
	if err != nil {
		return store.WrapTransactionError(err, "begin")
	}

	// Create transaction info
	info := &TxInfo{
		ReadOnly:  opts.ReadOnly,
		StartTime: time.Now(),
		Options:   opts,
	}

	// Add transaction and info to context
	ctxWithTx := context.WithValue(ctx, txContextKey{}, tx)
	ctxWithInfo := context.WithValue(ctxWithTx, txInfoKey{}, info)

	// Execute function
	if err := fn(ctxWithInfo); err != nil {
		_ = tx.Rollback()
		return store.WrapTransactionError(err, "rollback")
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return store.WrapTransactionError(err, "commit")
	}

	return nil
}

func (t *TransactionHandler) withRetry(ctx context.Context, opts store.TxOptions, fn func(context.Context) error) error {
	retryPolicy := opts.RetryPolicy
	var lastErr error

	for attempt := 0; attempt <= retryPolicy.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			delay := time.Duration(float64(retryPolicy.InitialDelay) * math.Pow(retryPolicy.BackoffMultiplier, float64(attempt-1)))
			if delay > retryPolicy.MaxDelay {
				delay = retryPolicy.MaxDelay
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue with retry
			}
		}

		err := t.executeTx(ctx, opts, fn)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable (implementation-specific)
		if !t.isRetryableError(err) {
			break
		}
	}

	return lastErr
}

func (t *TransactionHandler) toSQLTxOptions(opts store.TxOptions) *sql.TxOptions {
	sqlOpts := t.adapter.DefaultTxOptions()
	if sqlOpts == nil {
		sqlOpts = &sql.TxOptions{}
	}

	// Apply read-only setting
	if opts.ReadOnly {
		ro := *sqlOpts
		ro.ReadOnly = true
		sqlOpts = &ro
	}

	// Apply isolation level
	if opts.Isolation != store.IsolationDefault {
		iso := *sqlOpts
		iso.Isolation = t.toSQLIsolationLevel(opts.Isolation)
		sqlOpts = &iso
	}

	return sqlOpts
}

func (t *TransactionHandler) toSQLIsolationLevel(level store.IsolationLevel) sql.IsolationLevel {
	switch level {
	case store.IsolationReadUncommitted:
		return sql.LevelReadUncommitted
	case store.IsolationReadCommitted:
		return sql.LevelReadCommitted
	case store.IsolationRepeatableRead:
		return sql.LevelRepeatableRead
	case store.IsolationSerializable:
		return sql.LevelSerializable
	default:
		return sql.LevelDefault
	}
}

func (t *TransactionHandler) isRetryableError(err error) bool {
	// This is database-specific logic
	// For now, implement basic retry logic for common conflict errors
	if store.IsTransactionError(err) {
		return true
	}

	// Check for specific SQL error codes that indicate conflicts
	// This would be enhanced per database adapter
	errMsg := err.Error()

	// Common conflict indicators
	retryablePatterns := []string{
		"serialization failure",
		"deadlock",
		"lock wait timeout",
		"could not serialize",
	}

	for _, pattern := range retryablePatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// Helper function
func contains(s, substr string) bool {
	return len(substr) <= len(s) && (len(substr) == 0 || s[len(s)-len(substr):] == substr ||
		(len(s) > len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
