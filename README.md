![](https://github.com/The-Synergy-Framework/media-assets/blob/main/store_logo.png)

## Store - Persistent Storage Framework

Multi-backend persistent storage with repository patterns and cursor-based pagination for the Synergy Framework.

- **Multi-backend support**: SQL (PostgreSQL, MySQL, SQLite), KV (Redis, Memory), Document stores
- **Repository pattern**: Clean abstraction over data access with entity-first design
- **Cursor-based pagination**: High-performance, consistent pagination for large datasets
- **File storage**: Backend-agnostic file storage (S3, local filesystem, IPFS)
- **Production-ready**: Connection pooling, metrics, timeouts, and error handling

### What's inside

- `store`: Core interfaces (`Service`, `Repository`, `EntityRepository`, `Adapter`, `Connection`), types, and base implementations
- `store/sql`: SQL database support with query builders and transactions
  - `store/sql/adapter`: Database adapters (PostgreSQL, MySQL, SQLite)
  - `store/sql/repository`: SQL-specific repository implementation
  - `store/sql/query`: Query builders (SELECT, INSERT, UPDATE, DELETE)
  - `store/sql/pagination`: SQL-specific cursor pagination
- `store/kv`: Key-value store support with pattern matching and expiration
  - `store/kv/adapter`: KV adapters (Memory, Redis, Etcd)
  - `store/kv/repository`: KV-specific repository implementation
- `store/files`: File storage abstraction with repository pattern
  - `store/files/adapter`: File storage adapters (filesystem, planned: S3, IPFS)
  - `store/files/repository`: High-level repository for file operations

## Quick Start

### SQL with Environment Configuration

The simplest way to get started is using environment variables:

```go
package main

import (
	"context"
	"log"
	"os"

	"core/entity"
	"store"
	sqlstore "store/sql"
)

func main() {
	// Set environment variables
	os.Setenv("DB_TYPE", "postgres")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USERNAME", "postgres")
	os.Setenv("DB_PASSWORD", "password")
	os.Setenv("DB_NAME", "myapp")

	ctx := context.Background()

	// Create service from environment
	svc, err := sqlstore.OpenFromEnv(ctx)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer svc.Close()

	// Create repository and use transactions
	userRepo := svc.NewRepository(&User{})
	err = store.RunTx(ctx, svc.Transactor(), func(ctx context.Context) error {
		user := &User{
			BaseEntity: entity.NewBaseEntity(),
			Name:       "John Doe",
			Email:      "john@example.com",
		}
		return userRepo.Create(ctx, user)
	})
	if err != nil {
		log.Fatalf("transaction failed: %v", err)
	}
}
```

### Files with Environment Configuration

```go
package main

import (
	"context"
	"log"
	"os"
	"strings"

	"store"
	"store/files/adapter"
)

func main() {
	// Set environment variables
	os.Setenv("FILES_ROOT", "./uploads")
	os.Setenv("FILES_MAX_SIZE", "10485760") // 10MB

	ctx := context.Background()

	// Create files repository from environment
	repo, err := adapter.OpenRepositoryFromEnv()
	if err != nil {
		log.Fatalf("failed to open files: %v", err)
	}

	// Save a file
	content := "Hello, World!"
	id, metadata, err := repo.SaveBytes(ctx, "hello.txt", []byte(content), "text/plain")
	if err != nil {
		log.Fatalf("failed to save file: %v", err)
	}
	log.Printf("saved file %s: %+v", id, metadata)

	// List files with pagination
	params := store.CursorParams{PageSize: 10}
	result, err := repo.List(ctx, params)
	if err != nil {
		log.Fatalf("failed to list files: %v", err)
	}
	log.Printf("found %d files", len(result.Items))
}
```

### Manual Configuration (SQL + Repository Pattern)

```go
package main

import (
	"context"
	"log"
	"time"

	"core/entity"
	"store"
	"store/sql"
	"store/sql/adapter"
)

// Example entity
type User struct {
	*entity.BaseEntity
	Name  string `json:"name" db:"name"`
	Email string `json:"email" db:"email"`
}

func (u *User) GetID() string { return u.BaseEntity.ID }
func (u *User) SetID(id string) { u.BaseEntity.ID = id }
func (u *User) GetCreatedAt() time.Time { return u.BaseEntity.CreatedAt }
func (u *User) SetCreatedAt(t time.Time) { u.BaseEntity.CreatedAt = t }
func (u *User) GetUpdatedAt() time.Time { return u.BaseEntity.UpdatedAt }
func (u *User) SetUpdatedAt(t time.Time) { u.BaseEntity.UpdatedAt = t }

func main() {
	ctx := context.Background()

	// 1) Create SQL service with PostgreSQL adapter
	config := &adapter.Config{
		Host:     "localhost",
		Port:     5432,
		Database: "myapp",
		Username: "postgres",
		Password: "password",
		SSL:      "disable",
		Pool: adapter.PoolConfig{
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
	}

	service, err := sqlstore.Open(ctx, adapter.PostgreSQL, config)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer service.Close()

	// 2) Create repository for User entity
	userRepo := service.Repository(&User{})

	// 3) CRUD operations
	user := &User{
		BaseEntity: entity.NewBaseEntity(),
		Name:       "John Doe",
		Email:      "john@example.com",
	}

	// Create
	if err := userRepo.Create(ctx, user); err != nil {
		log.Fatalf("failed to create user: %v", err)
	}
	log.Printf("created user: %s", user.ID)

	// Read
	retrieved, err := userRepo.GetByID(ctx, user.ID)
	if err != nil {
		log.Fatalf("failed to get user: %v", err)
	}
	log.Printf("retrieved user: %s", retrieved.GetID())

	// Update
	user.Name = "Jane Doe"
	if err := userRepo.Update(ctx, user); err != nil {
		log.Fatalf("failed to update user: %v", err)
	}

	// Delete
	if err := userRepo.DeleteByID(ctx, user.ID); err != nil {
		log.Fatalf("failed to delete user: %v", err)
	}
}
```

### Cursor-Based Pagination Example

```go
// List users with cursor-based pagination
func listUsers(ctx context.Context, repo store.EntityRepository[User], pageSize int32, cursor string) {
	// Parse pagination parameters
	paginator := store.NewPaginator()
	params := paginator.ParseParams(pageSize, cursor)

	// Execute paginated query
	result, err := repo.List(ctx, params.PageSize, params.Cursor)
	if err != nil {
		log.Fatalf("failed to list users: %v", err)
	}

	// Process results
	for _, user := range result.Items {
		log.Printf("user: %s (%s)", user.Name, user.Email)
	}

	// Check if there are more pages
	if result.HasMore {
		log.Printf("next cursor: %s", result.NextCursor)
	}

	// Get page info
	info := paginator.GetPageInfo(nil, params.PageSize, result.TotalCount)
	log.Printf("page info: %+v", info)
}
```

### KV Store Example

```go
package main

import (
	"context"
	"log"
	"time"

	"core/entity"
	"store"
	"store/kv"
	"store/kv/adapter"
)

func main() {
	ctx := context.Background()

	// 1) Create KV service with memory adapter
	config := &adapter.Config{
		EnableMetrics: true,
		MetricLabels: map[string]string{
			"environment": "development",
		},
	}

	service, err := kvstore.Open(ctx, adapter.Memory, config)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer service.Close()

	// 2) Create repository for User entity
	userRepo := service.Repository(&User{})

	// 3) KV-specific operations
	user := &User{
		BaseEntity: entity.NewBaseEntity(),
		Name:       "John Doe",
		Email:      "john@example.com",
	}

	// Set with TTL
	if err := userRepo.SetWithTTL(ctx, user, 1*time.Hour); err != nil {
		log.Fatalf("failed to set user: %v", err)
	}

	// Get with TTL
	retrieved, ttl, err := userRepo.GetWithTTL(ctx, user.ID)
	if err != nil {
		log.Fatalf("failed to get user: %v", err)
	}
	log.Printf("user: %s, TTL: %v", retrieved.Name, ttl)

	// Pattern-based listing
	users, cursor, err := userRepo.ListByPattern(ctx, "*doe*", 10, "")
	if err != nil {
		log.Fatalf("failed to list users: %v", err)
	}
	log.Printf("found %d users matching pattern", len(users))

	// Batch operations
	usersBatch := []*User{
		{BaseEntity: entity.NewBaseEntity(), Name: "Alice", Email: "alice@example.com"},
		{BaseEntity: entity.NewBaseEntity(), Name: "Bob", Email: "bob@example.com"},
	}

	if err := userRepo.SetBatch(ctx, usersBatch, 24*time.Hour); err != nil {
		log.Fatalf("failed to set batch: %v", err)
	}
}
```

### File Storage Example

```go
package main

import (
	"context"
	"log"
	"strings"
	"time"

	"store"
	"store/files/adapter"
)

func main() {
	ctx := context.Background()

	// 1) Create filesystem repository
	config := adapter.FilesystemConfig{
		Root:      "/tmp/store",
		BaseURL:   "http://localhost:8080/files",
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		ChunkSize:   2 * 1024 * 1024,  // 2MB chunks
	}

	repo, err := adapter.OpenRepository(config)
	if err != nil {
		log.Fatalf("failed to create files repository: %v", err)
	}

	// 2) Save a file from content
	content := strings.NewReader("This is a test document content")
	fileID, metadata, err := repo.Save(ctx, "document.txt", content, "text/plain")
	if err != nil {
		log.Fatalf("failed to save file: %v", err)
	}
	log.Printf("saved file %s: %+v", fileID, metadata)

	// 3) Get file content
	reader, metadata, err := repo.Get(ctx, fileID)
	if err != nil {
		log.Fatalf("failed to get file: %v", err)
	}
	defer reader.Close()
	log.Printf("file metadata: %+v", metadata)

	// 4) List files with pagination
	params := store.CursorParams{PageSize: 10}
	result, err := repo.List(ctx, params)
	if err != nil {
		log.Fatalf("failed to list files: %v", err)
	}
	log.Printf("found %d files", len(result.Items))

	// 5) Generate presigned URL for download
	url, err := repo.PresignedURL(ctx, fileID, 1*time.Hour)
	if err != nil {
		log.Fatalf("failed to generate presigned URL: %v", err)
	}
	log.Printf("download URL: %s", url)

	// 6) Clean up
	if err := repo.Delete(ctx, fileID); err != nil {
		log.Fatalf("failed to delete file: %v", err)
	}
}
```

### Advanced Usage

#### Custom Query Building (SQL)

```go
// Build complex queries
qb := sqlstore.NewQueryBuilder("users").
	Select("id", "name", "email").
	Where("created_at", ">=", time.Now().AddDate(0, -1, 0)).
	Where("status", "=", "active").
	OrderBy("created_at", "DESC").
	Limit(100)

// Execute with pagination
result, err := sqlstore.ExecutePaginatedQuery(ctx, paginator, queryExecutor, qb, params, scanFunc)
```

#### Transaction Support

```go
// Execute operations in transaction
err := service.TransactionHandler().WithTransaction(ctx, func(tx *sql.Tx) error {
	// Create user
	if err := userRepo.Create(ctx, user); err != nil {
		return err
	}

	// Create user profile
	profile := &UserProfile{UserID: user.ID, Bio: "Hello world"}
	if err := profileRepo.Create(ctx, profile); err != nil {
		return err
	}

	return nil
})
```

#### Metrics and Observability

```go
// Enable metrics
config := &adapter.Config{
	EnableMetrics: true,
	MetricLabels: map[string]string{
		"environment": "production",
		"region":      "us-west-2",
	},
}

// Get connection stats
stats := service.Stats()
log.Printf("connection stats: %+v", stats)
```

### Configuration

#### SQL Configuration

```go
type Config struct {
	// Connection
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSL      string

	// Pool settings
	Pool PoolConfig

	// Timeouts
	ConnectTimeout time.Duration
	QueryTimeout   time.Duration

	// Observability
	EnableMetrics bool
	MetricLabels  map[string]string
}
```

#### KV Configuration

```go
type Config struct {
	// Connection
	Host     string
	Port     int
	Password string
	Database int

	// Timeouts
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration

	// Observability
	EnableMetrics bool
	MetricLabels  map[string]string
}
```

### Error Handling

The store library provides comprehensive error handling with typed errors:

```go
import "store"

// Check error types
if store.IsConnectionError(err) {
	// Handle connection issues
}

if store.IsRecordNotFoundError(err) {
	// Handle missing records
}

if store.IsTransactionError(err) {
	// Handle transaction failures
}

// Wrap errors with context
wrappedErr := store.WrapQueryError(err, "create", "users", "user-123", []any{user})
```

### Performance Features

- **Connection pooling**: Configurable connection pools for SQL databases
- **Cursor-based pagination**: Consistent performance regardless of page number
- **Batch operations**: Efficient bulk operations for KV stores
- **Lazy loading**: Entity scanning only when needed
- **Pattern matching**: Efficient key pattern searches in KV stores

### Contributing

The store library follows the Synergy Framework's design principles:

1. **Entity-first design**: All storage operations work with domain entities
2. **Backend agnostic**: Core interfaces don't leak backend-specific details
3. **Adapter pattern**: Pluggable backends through common interfaces
4. **Repository pattern**: Clean abstraction over data access
5. **Production ready**: Built-in observability, error handling, and performance features

### License

This project is part of the Synergy Framework and follows the same licensing terms.