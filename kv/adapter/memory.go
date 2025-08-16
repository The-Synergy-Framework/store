package adapter

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MemoryAdapter implements the Adapter interface using in-memory storage.
type MemoryAdapter struct {
	store *MemoryStore
}

// MemoryStore represents an in-memory key-value store.
type MemoryStore struct {
	mu    sync.RWMutex
	data  map[string]*MemoryValue
	stats *MemoryStats
}

// MemoryValue represents a value in memory with expiration.
type MemoryValue struct {
	Data      []byte
	ExpiresAt *time.Time
}

// MemoryStats tracks memory store statistics.
type MemoryStats struct {
	Keys         int64
	Gets         int64
	Sets         int64
	Deletes      int64
	Hits         int64
	Misses       int64
	Expired      int64
	LastAccessed time.Time
}

// MemoryConnection implements the Connection interface for memory storage.
type MemoryConnection struct {
	store *MemoryStore
}

// NewMemoryAdapter creates a new memory adapter.
func NewMemoryAdapter() *MemoryAdapter {
	return &MemoryAdapter{
		store: &MemoryStore{
			data:  make(map[string]*MemoryValue),
			stats: &MemoryStats{},
		},
	}
}

// Name returns the adapter name.
func (a *MemoryAdapter) Name() string {
	return "memory"
}

// Connect establishes a connection to memory storage.
func (a *MemoryAdapter) Connect(ctx context.Context, config *Config) (Connection, error) {
	return &MemoryConnection{store: a.store}, nil
}

// ConnectionString returns a memory connection string.
func (a *MemoryAdapter) ConnectionString(config *Config) string {
	return "memory://localhost"
}

// Store capabilities
func (a *MemoryAdapter) SupportsExpiration() bool      { return true }
func (a *MemoryAdapter) SupportsTransactions() bool    { return false } // Simplified for now
func (a *MemoryAdapter) SupportsPipelining() bool      { return false } // Simplified for now
func (a *MemoryAdapter) SupportsPatternMatching() bool { return true }
func (a *MemoryAdapter) SupportsPubSub() bool          { return false }

// Data type support
func (a *MemoryAdapter) SupportsLists() bool      { return false }
func (a *MemoryAdapter) SupportsSets() bool       { return false }
func (a *MemoryAdapter) SupportsHashes() bool     { return false }
func (a *MemoryAdapter) SupportsSortedSets() bool { return false }
func (a *MemoryAdapter) SupportsStreams() bool    { return false }

// Error classification
func (a *MemoryAdapter) IsKeyNotFoundError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "key not found")
}

func (a *MemoryAdapter) IsConnectionError(err error) bool {
	return false // Memory adapter doesn't have connection errors
}

func (a *MemoryAdapter) IsTimeoutError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "timeout")
}

// Close releases resources.
func (a *MemoryAdapter) Close() error {
	a.store.mu.Lock()
	defer a.store.mu.Unlock()

	// Clear all data
	a.store.data = make(map[string]*MemoryValue)
	a.store.stats = &MemoryStats{}

	return nil
}

// MemoryConnection implementations

// Get retrieves a value by key.
func (c *MemoryConnection) Get(ctx context.Context, key string) ([]byte, error) {
	c.store.mu.RLock()
	defer c.store.mu.RUnlock()

	c.store.stats.Gets++
	c.store.stats.LastAccessed = time.Now()

	value, exists := c.store.data[key]
	if !exists {
		c.store.stats.Misses++
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// Check expiration
	if value.ExpiresAt != nil && time.Now().After(*value.ExpiresAt) {
		delete(c.store.data, key)
		c.store.stats.Keys--
		c.store.stats.Expired++
		c.store.stats.Misses++
		return nil, fmt.Errorf("key not found: %s", key)
	}

	c.store.stats.Hits++
	return value.Data, nil
}

// Set stores a value with optional expiration.
func (c *MemoryConnection) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()

	c.store.stats.Sets++
	c.store.stats.LastAccessed = time.Now()

	var expiresAt *time.Time
	if expiration > 0 {
		expires := time.Now().Add(expiration)
		expiresAt = &expires
	}

	// Check if key already exists
	if _, exists := c.store.data[key]; !exists {
		c.store.stats.Keys++
	}

	c.store.data[key] = &MemoryValue{
		Data:      value,
		ExpiresAt: expiresAt,
	}

	return nil
}

// Delete removes a key.
func (c *MemoryConnection) Delete(ctx context.Context, key string) error {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()

	c.store.stats.Deletes++
	c.store.stats.LastAccessed = time.Now()

	if _, exists := c.store.data[key]; exists {
		delete(c.store.data, key)
		c.store.stats.Keys--
	}

	return nil
}

// Exists checks if a key exists.
func (c *MemoryConnection) Exists(ctx context.Context, key string) (bool, error) {
	c.store.mu.RLock()
	defer c.store.mu.RUnlock()

	value, exists := c.store.data[key]
	if !exists {
		return false, nil
	}

	// Check expiration
	if value.ExpiresAt != nil && time.Now().After(*value.ExpiresAt) {
		delete(c.store.data, key)
		c.store.stats.Keys--
		c.store.stats.Expired++
		return false, nil
	}

	return true, nil
}

// Batch operations (simplified implementations)
func (c *MemoryConnection) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	for _, key := range keys {
		if value, err := c.Get(ctx, key); err == nil {
			result[key] = value
		}
	}
	return result, nil
}

func (c *MemoryConnection) MSet(ctx context.Context, pairs map[string][]byte, expiration time.Duration) error {
	for key, value := range pairs {
		if err := c.Set(ctx, key, value, expiration); err != nil {
			return err
		}
	}
	return nil
}

func (c *MemoryConnection) MDelete(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := c.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// Pattern operations
func (c *MemoryConnection) Keys(ctx context.Context, pattern string) ([]string, error) {
	c.store.mu.RLock()
	defer c.store.mu.RUnlock()

	var keys []string
	for key := range c.store.data {
		if matchPattern(key, pattern) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func (c *MemoryConnection) Scan(ctx context.Context, cursor string, pattern string, count int) ([]string, string, error) {
	keys, err := c.Keys(ctx, pattern)
	if err != nil {
		return nil, "", err
	}

	// Simple pagination implementation
	start := 0
	if cursor != "" {
		// Parse cursor (simplified)
		for i, key := range keys {
			if key == cursor {
				start = i + 1
				break
			}
		}
	}

	end := start + count
	if end > len(keys) {
		end = len(keys)
	}

	var nextCursor string
	if end < len(keys) {
		nextCursor = keys[end-1]
	}

	return keys[start:end], nextCursor, nil
}

// Expiration operations
func (c *MemoryConnection) Expire(ctx context.Context, key string, expiration time.Duration) error {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()

	value, exists := c.store.data[key]
	if !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	expires := time.Now().Add(expiration)
	value.ExpiresAt = &expires

	return nil
}

func (c *MemoryConnection) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.store.mu.RLock()
	defer c.store.mu.RUnlock()

	value, exists := c.store.data[key]
	if !exists {
		return 0, fmt.Errorf("key not found: %s", key)
	}

	if value.ExpiresAt == nil {
		return -1, nil // No expiration
	}

	ttl := time.Until(*value.ExpiresAt)
	if ttl < 0 {
		return 0, nil // Expired
	}

	return ttl, nil
}

// Atomic operations
func (c *MemoryConnection) Incr(ctx context.Context, key string) (int64, error) {
	return c.IncrBy(ctx, key, 1)
}

func (c *MemoryConnection) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()

	// Simplified implementation - would need proper integer handling
	return value, fmt.Errorf("atomic operations not fully implemented in memory adapter")
}

func (c *MemoryConnection) Decr(ctx context.Context, key string) (int64, error) {
	return c.DecrBy(ctx, key, 1)
}

func (c *MemoryConnection) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.IncrBy(ctx, key, -value)
}

// Transaction and Pipeline support (not implemented for memory)
func (c *MemoryConnection) Pipeline() Pipeline {
	return nil // Not implemented
}

func (c *MemoryConnection) Transaction() Transaction {
	return nil // Not implemented
}

// Health and stats
func (c *MemoryConnection) Ping(ctx context.Context) error {
	return nil // Always healthy for memory
}

func (c *MemoryConnection) Stats() interface{} {
	c.store.mu.RLock()
	defer c.store.mu.RUnlock()

	return *c.store.stats
}

func (c *MemoryConnection) Close() error {
	return nil // Nothing to close for memory
}

// Helper function for pattern matching (simplified glob-style)
func matchPattern(key, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Simple prefix matching for now
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(key, prefix)
	}

	return key == pattern
}
