// Package store provides a persistent storage framework with multi-driver support
// and repository patterns for the Synergy Framework.
//
// This package follows the same architectural patterns as the guard package,
// with core abstractions at the root level and backend-specific implementations
// in sub-packages.
package store

import (
	"context"
	"time"

	"core/entity"
)

// Service defines the common interface for all storage services.
// Different backends (SQL, KV, Document) implement this interface.
type Service interface {
	// Connect establishes the connection to the storage backend
	Connect(ctx context.Context) error

	// Close closes the connection and releases resources
	Close() error

	// Stats returns backend-specific statistics
	Stats() interface{}

	// NewRepository creates a new repository for the given entity type
	NewRepository(entity entity.Entity) Repository

	// WithTimeout creates a context with timeout for operations
	WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc)
}

// Transactor provides a backend-agnostic transaction execution contract.
// Implementations may be no-ops if the backend does not support transactions.
type Transactor interface {
	// WithTx executes fn within a read-write transaction when supported.
	// The provided context may carry a backend-specific transaction handle.
	WithTx(ctx context.Context, fn func(context.Context) error) error

	// WithReadTx executes fn within a read-only transaction when supported.
	WithReadTx(ctx context.Context, fn func(context.Context) error) error

	// WithTxOptions executes fn within a transaction with specific options.
	WithTxOptions(ctx context.Context, opts TxOptions, fn func(context.Context) error) error

	// HasTx returns true if the context contains an active transaction.
	HasTx(ctx context.Context) bool

	// IsTxReadOnly returns true if the current transaction is read-only.
	IsTxReadOnly(ctx context.Context) bool
}

// TxOptions configures transaction behavior across different backends.
type TxOptions struct {
	// ReadOnly specifies if the transaction should be read-only
	ReadOnly bool

	// Isolation level (implementation-specific)
	Isolation IsolationLevel

	// Timeout for the transaction
	Timeout time.Duration

	// RetryPolicy defines retry behavior on transaction conflicts
	RetryPolicy *RetryPolicy

	// Backend-specific options
	BackendOptions map[string]any
}

// IsolationLevel represents transaction isolation levels.
type IsolationLevel string

const (
	IsolationDefault         IsolationLevel = "default"
	IsolationReadUncommitted IsolationLevel = "read_uncommitted"
	IsolationReadCommitted   IsolationLevel = "read_committed"
	IsolationRepeatableRead  IsolationLevel = "repeatable_read"
	IsolationSerializable    IsolationLevel = "serializable"
)

// RetryPolicy defines how transactions should be retried on conflicts.
type RetryPolicy struct {
	MaxRetries        int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
}

// DefaultRetryPolicy returns a sensible default retry policy.
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:        3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// TransactionManager provides advanced transaction management capabilities.
type TransactionManager interface {
	Transactor

	// Savepoint creates a savepoint within the current transaction (if supported).
	Savepoint(ctx context.Context, name string) error

	// RollbackToSavepoint rolls back to a specific savepoint.
	RollbackToSavepoint(ctx context.Context, name string) error

	// ReleaseSavepoint releases a savepoint.
	ReleaseSavepoint(ctx context.Context, name string) error
}

// Connection represents a generic connection interface.
type Connection interface {
	Ping(ctx context.Context) error
	Close() error
	Stats() interface{}
}

// Adapter represents a generic adapter interface.
type Adapter interface {
	Name() string
	Connect(ctx context.Context, config *Config) (Connection, error)
	Close() error
}

// Registry defines the interface for adapter registries.
type Registry interface {
	Get(name string) (Adapter, error)
	Register(name string, factory func() Adapter)
	List() []string
	Exists(name string) bool
}

// File represents a file with its content and metadata.
type File interface {
	ID() FileID
	Name() string
	Size() int64
	ContentType() string
	Content() []byte
	Metadata() map[string]string
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

// FileID represents a unique file identifier.
type FileID string

// String returns the string representation of the FileID.
func (id FileID) String() string {
	return string(id)
}

// IsEmpty returns true if the FileID is empty.
func (id FileID) IsEmpty() bool {
	return string(id) == ""
}

// FileMetadata contains file information without the actual content.
type FileMetadata struct {
	ID          FileID            `json:"id"`
	Name        string            `json:"name"`
	Size        int64             `json:"size"`
	ContentType string            `json:"content_type"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	StoredAt    string            `json:"stored_at"` // Backend-specific location
}

// BasicFile provides a simple implementation of the File interface.
type BasicFile struct {
	id          FileID
	name        string
	size        int64
	contentType string
	content     []byte
	metadata    map[string]string
	createdAt   time.Time
	updatedAt   time.Time
}

// NewBasicFile creates a new BasicFile.
func NewBasicFile(name string, content []byte, contentType string) *BasicFile {
	now := time.Now()
	return &BasicFile{
		id:          FileID(generateFileID(name, content)),
		name:        name,
		size:        int64(len(content)),
		contentType: contentType,
		content:     content,
		metadata:    make(map[string]string),
		createdAt:   now,
		updatedAt:   now,
	}
}

// File interface implementation
func (f *BasicFile) ID() FileID                  { return f.id }
func (f *BasicFile) Name() string                { return f.name }
func (f *BasicFile) Size() int64                 { return f.size }
func (f *BasicFile) ContentType() string         { return f.contentType }
func (f *BasicFile) Content() []byte             { return f.content }
func (f *BasicFile) Metadata() map[string]string { return f.metadata }
func (f *BasicFile) CreatedAt() time.Time        { return f.createdAt }
func (f *BasicFile) UpdatedAt() time.Time        { return f.updatedAt }

// SetMetadata sets a metadata key-value pair.
func (f *BasicFile) SetMetadata(key, value string) {
	if f.metadata == nil {
		f.metadata = make(map[string]string)
	}
	f.metadata[key] = value
	f.updatedAt = time.Now()
}

// generateFileID generates a unique file ID based on name and content.
func generateFileID(name string, content []byte) string {
	// This is a simplified implementation
	// In production, you might want a more sophisticated ID generation
	return name + "-" + time.Now().Format("20060102150405")
}

// OpenFunc represents a function that opens a service with an adapter.
type OpenFunc[T Service] func(ctx context.Context, adapter Adapter, config *Config) (T, error)

// OpenWithNameFunc represents a function that opens a service by adapter name.
type OpenWithNameFunc[T Service] func(ctx context.Context, adapterName string, config *Config) (T, error)

// RunTx executes fn within a read-write transaction when supported.
// This is a convenience helper that delegates to the Transactor interface.
func RunTx(ctx context.Context, tx Transactor, fn func(context.Context) error) error {
	return tx.WithTx(ctx, fn)
}

// RunReadTx executes fn within a read-only transaction when supported.
// This is a convenience helper that delegates to the Transactor interface.
func RunReadTx(ctx context.Context, tx Transactor, fn func(context.Context) error) error {
	return tx.WithReadTx(ctx, fn)
}
