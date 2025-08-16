package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"store"
)

// MutationExecutor handles execution of compiled mutations for SQL databases.
type MutationExecutor struct {
	db *sql.DB
}

// NewMutationExecutor creates a new SQL mutation executor.
func NewMutationExecutor(db *sql.DB) *MutationExecutor {
	return &MutationExecutor{db: db}
}

// Execute executes a mutation and returns result metadata.
func (me *MutationExecutor) Execute(ctx context.Context, mutation store.Mutation) (store.MutationResult, error) {
	// For now, we need a table name - this would be provided by the repository
	// This is a placeholder implementation
	return store.MutationResult{}, store.NewValidationError("Execute requires table name, use ExecuteForTable")
}

// ExecuteCompiled executes a pre-compiled mutation.
func (me *MutationExecutor) ExecuteCompiled(ctx context.Context, compiled store.CompiledMutation) (store.MutationResult, error) {
	// For simplicity, we'll handle RETURNING clauses later
	// Right now, just do regular execution
	return me.executeRegular(ctx, compiled)
}

// ExecuteForTable executes a mutation for a specific table.
func (me *MutationExecutor) ExecuteForTable(ctx context.Context, table string, mutation store.Mutation) (store.MutationResult, error) {
	compiled, err := CompileMutation(table, mutation)
	if err != nil {
		return store.MutationResult{}, err
	}

	return me.ExecuteCompiled(ctx, *compiled)
}

// executeRegular executes a mutation without RETURNING clause.
func (me *MutationExecutor) executeRegular(ctx context.Context, compiled store.CompiledMutation) (store.MutationResult, error) {
	var result sql.Result
	var err error

	if tx, ok := TransactionFromContext(ctx); ok && tx != nil {
		result, err = tx.ExecContext(ctx, compiled.SQL, compiled.Args...)
	} else {
		result, err = me.db.ExecContext(ctx, compiled.SQL, compiled.Args...)
	}

	if err != nil {
		return store.MutationResult{}, err
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	return store.MutationResult{
		RowsAffected: rowsAffected,
		LastInsertID: fmt.Sprintf("%d", lastInsertID),
		Returning:    nil,
	}, nil
}

// Batch mutation operations

// ExecuteBatch executes multiple mutations in a single transaction.
func (me *MutationExecutor) ExecuteBatch(ctx context.Context, mutations []store.CompiledMutation) ([]store.MutationResult, error) {
	// If we're already in a transaction, execute directly
	if tx, ok := TransactionFromContext(ctx); ok && tx != nil {
		return me.executeBatchInTx(ctx, tx, mutations)
	}

	// Start a new transaction
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, store.WrapTransactionError(err, "begin_batch")
	}

	// Add transaction to context
	txCtx := context.WithValue(ctx, txContextKey{}, tx)

	results, err := me.executeBatchInTx(txCtx, tx, mutations)
	if err != nil {
		_ = tx.Rollback()
		return nil, store.WrapTransactionError(err, "rollback_batch")
	}

	if err = tx.Commit(); err != nil {
		return nil, store.WrapTransactionError(err, "commit_batch")
	}

	return results, nil
}

// executeBatchInTx executes multiple mutations within an existing transaction.
func (me *MutationExecutor) executeBatchInTx(ctx context.Context, tx *sql.Tx, mutations []store.CompiledMutation) ([]store.MutationResult, error) {
	results := make([]store.MutationResult, len(mutations))

	for i, mutation := range mutations {
		result, err := me.ExecuteCompiled(ctx, mutation)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}

	return results, nil
}

// Specialized mutation methods

// Insert executes an INSERT mutation.
func (me *MutationExecutor) Insert(ctx context.Context, table string, values map[string]any) (store.MutationResult, error) {
	mutation := store.Insert{Values: values}
	return me.ExecuteForTable(ctx, table, mutation)
}

// InsertWithReturning executes an INSERT mutation with RETURNING clause.
func (me *MutationExecutor) InsertWithReturning(ctx context.Context, table string, values map[string]any, returning []string) (store.MutationResult, error) {
	mutation := store.Insert{Values: values}.WithReturning(returning...)
	return me.ExecuteForTable(ctx, table, mutation)
}
