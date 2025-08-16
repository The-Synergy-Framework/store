package kvstore

import (
	"context"

	"core/entity"
	"store"
)

// Repository provides KV storage implementing the standardized interface.
type Repository struct {
	*store.RepositoryBase
	kvService *Service
	keyPrefix string
}

// Ensure Repository implements store.Repository
var _ store.Repository = (*Repository)(nil)

// NewRepository creates a new KV repository.
func NewRepository(service *Service, ent entity.Entity) *Repository {
	base := store.NewRepositoryBase(ent)
	keyPrefix := entity.GetEntityName(ent) + ":"

	return &Repository{
		RepositoryBase: base,
		kvService:      service,
		keyPrefix:      keyPrefix,
	}
}

// Core CRUD operations

// Create stores a new entity in the KV store.
func (r *Repository) Create(ctx context.Context, ent entity.Entity) error {
	if err := r.Validate(ctx, ent); err != nil {
		return err
	}

	r.SetTimestamps(ent, true)

	key := r.keyPrefix + ent.GetID()

	// Check if entity already exists
	exists, err := r.kvService.Exists(ctx, key)
	if err != nil {
		return r.HandleGetError(err, "exists_check", ent.GetID())
	}

	if exists {
		return store.NewValidationError("entity already exists: " + ent.GetID())
	}

	err = r.kvService.SetJSON(ctx, key, ent, 0) // No expiration by default
	if err != nil {
		return r.HandleUpdateError(err, "create", ent.GetID())
	}

	return nil
}

// Get retrieves an entity by ID.
func (r *Repository) Get(ctx context.Context, id string) (entity.Entity, error) {
	if err := r.ValidateID(id); err != nil {
		return nil, err
	}

	key := r.keyPrefix + id
	newEntity := r.CreateNewEntity()

	err := r.kvService.GetJSON(ctx, key, newEntity)
	if err != nil {
		if r.kvService.adapter.IsKeyNotFoundError(err) {
			return nil, store.NewRecordNotFoundError(r.EntityName(), id)
		}
		return nil, r.HandleGetError(err, "get", id)
	}

	return newEntity, nil
}

// Update modifies an existing entity in the KV store.
func (r *Repository) Update(ctx context.Context, ent entity.Entity) error {
	if err := r.Validate(ctx, ent); err != nil {
		return err
	}

	r.SetTimestamps(ent, false)

	key := r.keyPrefix + ent.GetID()

	// Check if entity exists
	exists, err := r.kvService.Exists(ctx, key)
	if err != nil {
		return r.HandleGetError(err, "exists_check", ent.GetID())
	}

	if !exists {
		return store.NewRecordNotFoundError(r.EntityName(), ent.GetID())
	}

	err = r.kvService.SetJSON(ctx, key, ent, 0)
	if err != nil {
		return r.HandleUpdateError(err, "update", ent.GetID())
	}

	return nil
}

// Delete removes an entity by ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	if err := r.ValidateID(id); err != nil {
		return err
	}

	key := r.keyPrefix + id

	err := r.kvService.Delete(ctx, key)
	if err != nil {
		if r.kvService.adapter.IsKeyNotFoundError(err) {
			return store.NewRecordNotFoundError(r.EntityName(), id)
		}
		return r.HandleUpdateError(err, "delete", id)
	}

	return nil
}

// Exists checks if an entity with the given ID exists.
func (r *Repository) Exists(ctx context.Context, id string) (bool, error) {
	if err := r.ValidateID(id); err != nil {
		return false, err
	}

	key := r.keyPrefix + id
	exists, err := r.kvService.Exists(ctx, key)
	if err != nil {
		return false, r.HandleGetError(err, "exists", id)
	}

	return exists, nil
}

// Batch operations

// CreateBatch creates multiple entities.
func (r *Repository) CreateBatch(ctx context.Context, entities []entity.Entity) error {
	for _, ent := range entities {
		if err := r.Create(ctx, ent); err != nil {
			return err
		}
	}
	return nil
}

// UpdateBatch updates multiple entities.
func (r *Repository) UpdateBatch(ctx context.Context, entities []entity.Entity) error {
	for _, ent := range entities {
		if err := r.Update(ctx, ent); err != nil {
			return err
		}
	}
	return nil
}

// DeleteBatch deletes multiple entities by IDs.
func (r *Repository) DeleteBatch(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := r.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// GetBatch retrieves multiple entities by IDs.
func (r *Repository) GetBatch(ctx context.Context, ids []string) (map[string]entity.Entity, error) {
	result := make(map[string]entity.Entity)

	for _, id := range ids {
		ent, err := r.Get(ctx, id)
		if err != nil {
			// Skip not found errors
			if !store.IsRecordNotFoundError(err) {
				return nil, err
			}
		} else {
			result[id] = ent
		}
	}

	return result, nil
}

// Query operations

// FindWhere returns entities matching the given conditions - limited support for KV stores.
func (r *Repository) FindWhere(ctx context.Context, conditions ...store.Condition) ([]entity.Entity, error) {
	// KV stores have limited query support - return empty for now
	// In a real implementation, this would require indexing or pattern matching
	return []entity.Entity{}, nil
}

// CountWhere returns the count of entities matching the given conditions - limited for KV stores.
func (r *Repository) CountWhere(ctx context.Context, conditions ...store.Condition) (int64, error) {
	// KV stores don't have efficient conditional counting - return 0 for now
	// In a real implementation, this would require indexing or scanning
	return 0, nil
}

// FindFirst returns the first entity matching the given conditions - limited for KV stores.
func (r *Repository) FindFirst(ctx context.Context, conditions ...store.Condition) (entity.Entity, error) {
	entities, err := r.FindWhere(ctx, conditions...)
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		return nil, store.NewRecordNotFoundError(r.EntityName(), "first")
	}
	return entities[0], nil
}

// List returns paginated results - simplified for KV stores.
func (r *Repository) List(ctx context.Context, params store.CursorParams) (store.CursorResult[entity.Entity], error) {
	// KV stores don't have efficient listing - return empty for now
	// In a real implementation, this would use pattern matching or indexing
	return store.CursorResult[entity.Entity]{
		Items:   []entity.Entity{},
		HasMore: false,
	}, nil
}

// Count returns the number of entities - limited for KV stores.
func (r *Repository) Count(ctx context.Context, conditions ...store.Condition) (int64, error) {
	// KV stores don't have efficient counting - return 0 for now
	// In a real implementation, this would require indexing or scanning
	return 0, nil
}

// HealthCheck performs a basic health check.
func (r *Repository) HealthCheck(ctx context.Context) error {
	// Simple existence check
	testKey := r.keyPrefix + "health_check"
	_, err := r.kvService.Exists(ctx, testKey)
	return err
}
