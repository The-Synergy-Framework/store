package store

import (
	"context"

	"core/entity"
)

// Repository defines the essential operations that all backends must implement.
// This provides a clean, consistent interface across all storage backends.
type Repository interface {
	EntityName() string

	Create(ctx context.Context, entity entity.Entity) error
	Get(ctx context.Context, id string) (entity.Entity, error)
	Update(ctx context.Context, entity entity.Entity) error
	Delete(ctx context.Context, id string) error
	Exists(ctx context.Context, id string) (bool, error)

	CreateBatch(ctx context.Context, entities []entity.Entity) error
	UpdateBatch(ctx context.Context, entities []entity.Entity) error
	DeleteBatch(ctx context.Context, ids []string) error
	GetBatch(ctx context.Context, ids []string) (map[string]entity.Entity, error)

	List(ctx context.Context, params CursorParams) (CursorResult[entity.Entity], error)
	FindWhere(ctx context.Context, conditions ...Condition) ([]entity.Entity, error)
	CountWhere(ctx context.Context, conditions ...Condition) (int64, error)

	FindFirst(ctx context.Context, conditions ...Condition) (entity.Entity, error)

	Validate(ctx context.Context, entity entity.Entity) error
	HealthCheck(ctx context.Context) error
}
