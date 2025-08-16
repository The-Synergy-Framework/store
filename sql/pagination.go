package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"store"
)

// SQLPaginator wraps the generic cursor paginator with SQL-specific functionality.
type SQLPaginator struct {
	*store.Paginator
}

// NewSQLPaginator creates a new SQL-specific paginator.
func NewSQLPaginator() *SQLPaginator {
	return &SQLPaginator{
		Paginator: store.NewPaginator(),
	}
}

// NewSQLPaginatorWithConfig creates a new SQL paginator with custom config.
func NewSQLPaginatorWithConfig(config store.PaginationConfig) *SQLPaginator {
	return &SQLPaginator{
		Paginator: store.NewPaginatorWithConfig(config),
	}
}

// ApplyToQueryBuilder applies cursor pagination parameters to a QueryBuilder using keyset pagination.
// Assumes ordering by created_at ASC, id ASC. Adjust as needed per repository.
func (p *SQLPaginator) ApplyToQueryBuilder(qb *QueryBuilder, params store.CursorParams) *QueryBuilder {
	qb = qb.Limit(int(params.PageSize))
	if params.Cursor == "" {
		return qb
	}
	cursor, err := p.DecodeCursor(params.Cursor)
	if err != nil || cursor == nil {
		return qb
	}
	if params.Backward {
		// Backward: (created_at < last_ts) OR (created_at = last_ts AND id < last_id)
		condA := Condition{Column: fmt.Sprintf("created_at < $%d", qb.argIndex)}
		qb.args = append(qb.args, cursor.LastTimestamp)
		qb.argIndex++
		condB := Condition{Column: fmt.Sprintf("(created_at = $%d AND id < $%d)", qb.argIndex, qb.argIndex+1)}
		qb.args = append(qb.args, cursor.LastTimestamp, cursor.LastID)
		qb.argIndex += 2
		qb.where = append(qb.where, Condition{Column: fmt.Sprintf("(%s OR %s)", condA.Column, condB.Column)})
		// Reverse order to fetch previous page items in correct sequence (ASC by created_at, id)
		qb.OrderByAsc("created_at").OrderByAsc("id")
		return qb
	}
	// Forward: (created_at > last_ts) OR (created_at = last_ts AND id > last_id)
	condA := Condition{Column: fmt.Sprintf("created_at > $%d", qb.argIndex)}
	qb.args = append(qb.args, cursor.LastTimestamp)
	qb.argIndex++
	condB := Condition{Column: fmt.Sprintf("(created_at = $%d AND id > $%d)", qb.argIndex, qb.argIndex+1)}
	qb.args = append(qb.args, cursor.LastTimestamp, cursor.LastID)
	qb.argIndex += 2
	qb.where = append(qb.where, Condition{Column: fmt.Sprintf("(%s OR %s)", condA.Column, condB.Column)})
	qb.OrderByAsc("created_at").OrderByAsc("id")
	return qb
}

// ExecutePaginatedQuery executes a cursor-based paginated query (keyset pagination).
func ExecutePaginatedQuery[T any](
	ctx context.Context,
	p *SQLPaginator,
	qe *QueryExecutor,
	qb *QueryBuilder,
	params store.CursorParams,
	scanFunc func(*sql.Rows) (T, error),
) (store.CursorResult[T], error) {
	paginatedQb := p.ApplyToQueryBuilder(qb, params)

	rows, err := qe.Query(ctx, paginatedQb)
	if err != nil {
		return store.CursorResult[T]{}, err
	}
	defer rows.Close()

	var items []T
	for rows.Next() {
		item, err := scanFunc(rows)
		if err != nil {
			return store.CursorResult[T]{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return store.CursorResult[T]{}, err
	}

	hasMore := int32(len(items)) == params.PageSize
	var totalCount int64 = -1
	if params.Cursor == "" {
		if count, err := qe.Count(ctx, qb); err == nil {
			totalCount = count
		}
	}

	result := store.BuildCursorResult(p.Paginator, items, params.PageSize, hasMore, totalCount)

	// Set PreviousCursor when paging backward or when a cursor is provided
	if params.Cursor != "" && len(items) > 0 {
		if prev, err := p.CreatePreviousCursor(items[0], params.PageSize); err == nil {
			if enc, err := p.EncodeCursor(prev); err == nil {
				result.PreviousCursor = enc
			}
		}
	}

	return result, nil
}
