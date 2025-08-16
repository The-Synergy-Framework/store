package sqlstore

import (
	"context"
	"database/sql"
	"store"

	"store/sql/adapter"
)

type txContextKey struct{}

// TransactionFromContext extracts an *sql.Tx from context when present.
func TransactionFromContext(ctx context.Context) (*sql.Tx, bool) {
	v := ctx.Value(txContextKey{})
	if v == nil {
		return nil, false
	}
	tx, ok := v.(*sql.Tx)
	return tx, ok
}

type TransactionHandler struct {
	db      *sql.DB
	adapter adapter.Adapter
}

func NewTransactionHandler(db *sql.DB, adpt adapter.Adapter) *TransactionHandler {
	return &TransactionHandler{db: db, adapter: adpt}
}

// Ensure TransactionHandler satisfies store.Transactor.
var _ store.Transactor = (*TransactionHandler)(nil)

func (t *TransactionHandler) WithTx(ctx context.Context, fn func(context.Context) error) error {
	// Reuse existing transaction if present
	if existing, ok := TransactionFromContext(ctx); ok && existing != nil {
		return fn(ctx)
	}

	tx, err := t.db.BeginTx(ctx, t.adapter.DefaultTxOptions())
	if err != nil {
		return store.WrapTransactionError(err, "begin")
	}
	ctxWithTx := context.WithValue(ctx, txContextKey{}, tx)
	if err := fn(ctxWithTx); err != nil {
		_ = tx.Rollback()
		return store.WrapTransactionError(err, "rollback")
	}
	if err := tx.Commit(); err != nil {
		return store.WrapTransactionError(err, "commit")
	}
	return nil
}

func (t *TransactionHandler) WithReadTx(ctx context.Context, fn func(context.Context) error) error {
	opts := t.adapter.DefaultTxOptions()
	if opts == nil {
		opts = &sql.TxOptions{}
	}
	ro := *opts
	ro.ReadOnly = true

	// Reuse existing transaction if present
	if existing, ok := TransactionFromContext(ctx); ok && existing != nil {
		return fn(ctx)
	}

	tx, err := t.db.BeginTx(ctx, &ro)
	if err != nil {
		return store.WrapTransactionError(err, "begin_read")
	}
	ctxWithTx := context.WithValue(ctx, txContextKey{}, tx)
	if err := fn(ctxWithTx); err != nil {
		_ = tx.Rollback()
		return store.WrapTransactionError(err, "rollback_read")
	}
	if err := tx.Commit(); err != nil {
		return store.WrapTransactionError(err, "commit_read")
	}
	return nil
}
