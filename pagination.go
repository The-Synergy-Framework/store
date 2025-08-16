package store

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// Cursor represents a pagination cursor that encodes position information.
// This provides consistent, performant pagination across large datasets.
type Cursor struct {
	// Position information
	LastID        string    `json:"id"`        // Last item ID from previous page
	LastTimestamp time.Time `json:"timestamp"` // Last item timestamp for ordering
	LastSort      string    `json:"sort"`      // Last item sort value (for custom ordering)

	// Metadata
	PageSize  int32     `json:"page_size"`  // Page size for this cursor
	CreatedAt time.Time `json:"created_at"` // When cursor was created
	Version   int       `json:"version"`    // Cursor format version
}

// CursorParams holds cursor-based pagination parameters.
type CursorParams struct {
	PageSize int32  // Number of items per page
	Cursor   string // Encoded cursor string (empty for first page)
}

// CursorResult holds the result of a cursor-based paginated query.
type CursorResult[T any] struct {
	Items          []T    // Items for current page
	NextCursor     string // Encoded cursor for next page (empty if no more pages)
	PreviousCursor string // Encoded cursor for previous page (empty if first page)
	HasMore        bool   // Whether there are more pages
	TotalCount     int64  // Total count (if available, may be -1 for unknown)
}

// PaginationConfig holds cursor pagination configuration.
type PaginationConfig struct {
	DefaultPageSize int32
	MaxPageSize     int32
	MinPageSize     int32
	MaxCursorAge    time.Duration // How long cursors remain valid
}

// DefaultPaginationConfig returns sensible cursor pagination defaults.
func DefaultPaginationConfig() PaginationConfig {
	return PaginationConfig{
		DefaultPageSize: 20,
		MaxPageSize:     100,
		MinPageSize:     1,
		MaxCursorAge:    24 * time.Hour, // Cursors expire after 24 hours
	}
}

// Paginator provides cursor-based pagination logic.
type Paginator struct {
	config PaginationConfig
}

// NewPaginator creates a new cursor paginator with default configuration.
func NewPaginator() *Paginator {
	return &Paginator{config: DefaultPaginationConfig()}
}

// NewPaginatorWithConfig creates a new cursor paginator with custom configuration.
func NewPaginatorWithConfig(config PaginationConfig) *Paginator {
	return &Paginator{config: config}
}

// ParseParams parses and validates cursor pagination parameters.
func (p *Paginator) ParseParams(pageSize int32, cursor string) CursorParams {
	// Validate and normalize page size
	if pageSize <= 0 {
		pageSize = p.config.DefaultPageSize
	}
	if pageSize > p.config.MaxPageSize {
		pageSize = p.config.MaxPageSize
	}
	if pageSize < p.config.MinPageSize {
		pageSize = p.config.MinPageSize
	}

	return CursorParams{
		PageSize: pageSize,
		Cursor:   cursor,
	}
}

// DecodeCursor decodes a cursor string into a Cursor struct.
func (p *Paginator) DecodeCursor(cursorStr string) (*Cursor, error) {
	if cursorStr == "" {
		return nil, nil
	}

	// Decode base64
	decoded, err := base64.URLEncoding.DecodeString(cursorStr)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}

	// Parse JSON
	var cursor Cursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, fmt.Errorf("invalid cursor content: %w", err)
	}

	// Validate cursor age
	if time.Since(cursor.CreatedAt) > p.config.MaxCursorAge {
		return nil, fmt.Errorf("cursor expired (age: %v, max: %v)",
			time.Since(cursor.CreatedAt), p.config.MaxCursorAge)
	}

	// Validate version compatibility
	if cursor.Version != 1 {
		return nil, fmt.Errorf("unsupported cursor version: %d", cursor.Version)
	}

	return &cursor, nil
}

// EncodeCursor encodes a Cursor struct into a base64 string.
func (p *Paginator) EncodeCursor(cursor *Cursor) (string, error) {
	if cursor == nil {
		return "", nil
	}

	// Set metadata
	if cursor.CreatedAt.IsZero() {
		cursor.CreatedAt = time.Now()
	}
	if cursor.Version == 0 {
		cursor.Version = 1
	}

	// Marshal to JSON
	data, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cursor: %w", err)
	}

	// Encode to base64
	return base64.URLEncoding.EncodeToString(data), nil
}

// CreateCursor creates a new cursor for the given item.
func (p *Paginator) CreateCursor(id string, timestamp time.Time, sortValue string, pageSize int32) *Cursor {
	return &Cursor{
		LastID:        id,
		LastTimestamp: timestamp,
		LastSort:      sortValue,
		PageSize:      pageSize,
		CreatedAt:     time.Now(),
		Version:       1,
	}
}

// CreateNextCursor creates a cursor for the next page.
func (p *Paginator) CreateNextCursor(lastItem interface{}, pageSize int32) (*Cursor, error) {
	// Try to extract ID and timestamp from the item
	id, timestamp, sortValue, err := p.extractItemInfo(lastItem)
	if err != nil {
		return nil, fmt.Errorf("failed to extract item info: %w", err)
	}

	return p.CreateCursor(id, timestamp, sortValue, pageSize), nil
}

// CreatePreviousCursor creates a cursor for the previous page.
func (p *Paginator) CreatePreviousCursor(firstItem interface{}, pageSize int32) (*Cursor, error) {
	// For previous page, we need the first item of the current page
	id, timestamp, sortValue, err := p.extractItemInfo(firstItem)
	if err != nil {
		return nil, fmt.Errorf("failed to extract item info: %w", err)
	}

	// Create a "reverse" cursor for previous page
	return &Cursor{
		LastID:        id,
		LastTimestamp: timestamp,
		LastSort:      sortValue,
		PageSize:      pageSize,
		CreatedAt:     time.Now(),
		Version:       1,
	}, nil
}

// extractItemInfo extracts ID, timestamp, and sort value from an item.
// This is a generic approach that can be overridden by specific implementations.
func (p *Paginator) extractItemInfo(item interface{}) (id string, timestamp time.Time, sortValue string, err error) {
	// Try to use reflection to get common fields
	// This is a fallback - specific repositories should override this
	switch v := item.(type) {
	case interface{ GetID() string }:
		id = v.GetID()
	case interface{ ID() string }:
		id = v.ID()
	default:
		id = fmt.Sprintf("%v", item)
	}

	// Try to get timestamp
	switch v := item.(type) {
	case interface{ GetCreatedAt() time.Time }:
		timestamp = v.GetCreatedAt()
	case interface{ CreatedAt() time.Time }:
		timestamp = v.CreatedAt()
	case interface{ GetUpdatedAt() time.Time }:
		timestamp = v.GetUpdatedAt()
	case interface{ UpdatedAt() time.Time }:
		timestamp = v.UpdatedAt()
	default:
		timestamp = time.Now()
	}

	// Sort value defaults to timestamp
	sortValue = timestamp.Format(time.RFC3339Nano)

	return id, timestamp, sortValue, nil
}

// BuildCursorResult creates a cursor result from items and metadata.
func BuildCursorResult[T any](
	p *Paginator,
	items []T,
	pageSize int32,
	hasMore bool,
	totalCount int64,
) CursorResult[T] {
	result := CursorResult[T]{
		Items:      items,
		HasMore:    hasMore,
		TotalCount: totalCount,
	}

	// Generate next cursor if there are more pages
	if hasMore && len(items) > 0 {
		if nextCursor, err := p.CreateNextCursor(items[len(items)-1], pageSize); err == nil {
			if encoded, err := p.EncodeCursor(nextCursor); err == nil {
				result.NextCursor = encoded
			}
		}
	}

	// Generate previous cursor if this isn't the first page
	// Note: This requires the original cursor to be available in the calling context
	// For now, we'll leave it empty and let the caller handle it

	return result
}

// ValidateCursor validates if a cursor string is valid.
func (p *Paginator) ValidateCursor(cursorStr string) error {
	_, err := p.DecodeCursor(cursorStr)
	return err
}

// GetPageInfo returns information about the current page (if available).
func (p *Paginator) GetPageInfo(cursor *Cursor, pageSize int32, totalCount int64) map[string]interface{} {
	info := map[string]interface{}{
		"page_size":   pageSize,
		"total_count": totalCount,
		"has_more":    totalCount > 0, // Simplified - actual logic depends on items
	}

	if cursor != nil {
		info["cursor_created"] = cursor.CreatedAt
		info["cursor_age"] = time.Since(cursor.CreatedAt).String()
	}

	return info
}

// Config returns the current pagination configuration.
func (p *Paginator) Config() PaginationConfig {
	return p.config
}

// Legacy support functions for backward compatibility

// LegacyOffsetParams converts cursor params to offset-based params (deprecated).
func (p *Paginator) LegacyOffsetParams(params CursorParams) (offset int, pageSize int32) {
	if params.Cursor == "" {
		return 0, params.PageSize
	}

	cursor, err := p.DecodeCursor(params.Cursor)
	if err != nil {
		return 0, params.PageSize
	}

	// This is approximate - cursor-based doesn't have exact offsets
	// Use timestamp-based approximation
	offset = int(time.Since(cursor.LastTimestamp).Seconds() / 60) // Rough estimate
	return offset, cursor.PageSize
}

// LegacyResult converts cursor result to legacy format (deprecated).
func LegacyResult[T any](cursorResult CursorResult[T]) map[string]interface{} {
	return map[string]interface{}{
		"items":           cursorResult.Items,
		"next_page_token": cursorResult.NextCursor,
		"total_count":     cursorResult.TotalCount,
		"has_more":        cursorResult.HasMore,
	}
}
