package sqlstore

import (
	"context"
	"database/sql"
	"store"

	"store/sql/adapter"
)

type TransactionHandler struct {
	db      *sql.DB
	adapter adapter.Adapter
}

func NewTransactionHandler(db *sql.DB, adpt adapter.Adapter) *TransactionHandler {
	return &TransactionHandler{db: db, adapter: adpt}
}

func (t *TransactionHandler) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := t.db.BeginTx(ctx, t.adapter.DefaultTxOptions())
	if err != nil {
		return store.WrapTransactionError(err, "begin")
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return store.WrapTransactionError(err, "rollback")
	}
	if err := tx.Commit(); err != nil {
		return store.WrapTransactionError(err, "commit")
	}
	return nil
}

func (t *TransactionHandler) WithReadTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	opts := t.adapter.DefaultTxOptions()
	if opts == nil {
		opts = &sql.TxOptions{}
	}
	ro := *opts
	ro.ReadOnly = true
	tx, err := t.db.BeginTx(ctx, &ro)
	if err != nil {
		return store.WrapTransactionError(err, "begin_read")
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return store.WrapTransactionError(err, "rollback_read")
	}
	if err := tx.Commit(); err != nil {
		return store.WrapTransactionError(err, "commit_read")
	}
	return nil
}
