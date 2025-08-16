package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

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

// ApplyToQueryBuilder applies cursor pagination parameters to a QueryBuilder.
// For SQL, we'll use LIMIT and WHERE clauses based on the cursor.
func (p *SQLPaginator) ApplyToQueryBuilder(qb *QueryBuilder, params store.CursorParams) *QueryBuilder {
	// Apply page size limit
	qb = qb.Limit(int(params.PageSize))

	// If we have a cursor, apply WHERE clause for cursor-based pagination
	if params.Cursor != "" {
		cursor, err := p.DecodeCursor(params.Cursor)
		if err == nil && cursor != nil {
			// Use the last item's timestamp and ID for cursor-based pagination
			// This assumes items are ordered by timestamp (created_at) and then by ID
			// For now, use a simple timestamp-based cursor until we implement compound cursors
			qb = qb.Where("created_at", "<", cursor.LastTimestamp)
		}
	}

	return qb
}

// ExecutePaginatedQuery executes a cursor-based paginated query.
func ExecutePaginatedQuery[T any](
	ctx context.Context,
	p *SQLPaginator,
	qe *QueryExecutor,
	qb *QueryBuilder,
	params store.CursorParams,
	scanFunc func(*sql.Rows) (T, error),
) (store.CursorResult[T], error) {
	// Apply pagination to the query builder
	paginatedQb := p.ApplyToQueryBuilder(qb, params)

	// Execute the query
	rows, err := qe.Query(ctx, paginatedQb)
	if err != nil {
		return store.CursorResult[T]{}, err
	}
	defer rows.Close()

	// Scan results using the provided scan function
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

	// Determine if there are more pages
	hasMore := len(items) == int(params.PageSize)

	// Get total count (without pagination) - this is optional for cursor-based pagination
	var totalCount int64 = -1 // -1 indicates unknown
	if params.Cursor == "" {
		// Only get total count for first page
		if count, err := qe.Count(ctx, qb); err == nil {
			totalCount = count
		}
	}

	// Build cursor result
	result := store.BuildCursorResult(p.Paginator, items, params.PageSize, hasMore, totalCount)
	return result, nil
}

// Legacy types for backward compatibility - these will be deprecated
type PaginationParams = store.CursorParams
type PaginationResult struct {
	Items         []interface{}
	NextPageToken string
	TotalCount    int64
	HasMore       bool
}

// Legacy constructor for backward compatibility
func NewPagination() *SQLPaginator {
	return NewSQLPaginator()
}

// Legacy method for backward compatibility
func (p *SQLPaginator) ExecutePaginatedQueryWithScan(
	ctx context.Context,
	qe *QueryExecutor,
	qb *QueryBuilder,
	params store.CursorParams,
	scanFunc func(*sql.Rows) (interface{}, error),
) (*PaginationResult, error) {
	result, err := ExecutePaginatedQuery(ctx, p, qe, qb, params, scanFunc)
	if err != nil {
		return nil, err
	}

	// Convert cursor result to legacy format
	legacyResult := &PaginationResult{
		Items:         make([]interface{}, len(result.Items)),
		NextPageToken: result.NextCursor,
		TotalCount:    result.TotalCount,
		HasMore:       result.HasMore,
	}

	// Convert items to interface{}
	for i, item := range result.Items {
		legacyResult.Items[i] = item
	}

	return legacyResult, nil
}

// LegacyParseParams converts legacy page token to cursor params (deprecated).
func (p *SQLPaginator) LegacyParseParams(pageSize int32, pageToken string) store.CursorParams {
	// For backward compatibility, try to parse old offset-based tokens
	if pageToken != "" {
		if offset, err := strconv.Atoi(pageToken); err == nil && offset >= 0 {
			// Convert offset to approximate cursor (this is not ideal but maintains compatibility)
			// In production, you should migrate to proper cursor-based pagination
			return store.CursorParams{
				PageSize: pageSize,
				Cursor:   fmt.Sprintf("legacy_offset_%d", offset),
			}
		}
	}

	return store.CursorParams{
		PageSize: pageSize,
		Cursor:   pageToken,
	}
}
