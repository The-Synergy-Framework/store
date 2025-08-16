package kvstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"core/entity"
	"store"
)

// Repository provides KV storage for a specific entity type.
// It leverages core/entity for serialization and key generation, and extends the shared base.
type Repository struct {
	*store.RepositoryBase
	service   *Service
	keyPrefix string
}

// Ensure Repository satisfies store-agnostic contracts.
var _ store.EntityRepository[entity.Entity] = (*Repository)(nil)
var _ store.Countable = (*Repository)(nil)

// NewRepository creates a new entity-specific KV repository.
func NewRepository(service *Service, ent entity.Entity) *Repository {
	keyPrefix := entity.GetEntityName(ent) + ":"

	return &Repository{
		RepositoryBase: store.NewRepositoryBase(ent),
		service:        service,
		keyPrefix:      keyPrefix,
	}
}

// Entity-agnostic interface implementation

// GetByID retrieves an entity by ID (tech-agnostic signature).
func (r *Repository) GetByID(ctx context.Context, id string) (entity.Entity, error) {
	if err := r.ValidateID(id); err != nil {
		return nil, err
	}

	key := r.keyPrefix + id

	// Create new entity instance using base functionality
	newEntity := r.CreateNewEntity()

	// Get JSON data
	err := r.service.GetJSON(ctx, key, newEntity)
	if err != nil {
		if r.service.adapter.IsKeyNotFoundError(err) {
			return nil, store.NewRecordNotFoundError(r.EntityName(), id)
		}
		return nil, r.HandleGetError(err, "get", id)
	}

	return newEntity, nil
}

// Exists checks if an entity exists by ID.
func (r *Repository) Exists(ctx context.Context, id string) (bool, error) {
	if err := r.ValidateID(id); err != nil {
		return false, err
	}

	key := r.keyPrefix + id
	exists, err := r.service.Exists(ctx, key)
	if err != nil {
		return false, r.HandleGetError(err, "exists", id)
	}
	return exists, nil
}

// DeleteByID deletes an entity by ID.
func (r *Repository) DeleteByID(ctx context.Context, id string) error {
	if err := r.ValidateID(id); err != nil {
		return err
	}

	key := r.keyPrefix + id

	// Check if exists first
	exists, err := r.service.Exists(ctx, key)
	if err != nil {
		return r.HandleGetError(err, "exists_check", id)
	}

	if !exists {
		return store.NewRecordNotFoundError(r.EntityName(), id)
	}

	err = r.service.Delete(ctx, key)
	if err != nil {
		return r.HandleUpdateError(err, "delete", id)
	}

	return nil
}

// Count returns the total count of entities (approximate for KV stores).
func (r *Repository) Count(ctx context.Context) (int64, error) {
	pattern := r.keyPrefix + "*"
	keys, err := r.service.Keys(ctx, pattern)
	if err != nil {
		return 0, r.HandleGetError(err, "count", "")
	}

	return int64(len(keys)), nil
}

// KV-specific methods

// Set stores an entity with optional expiration.
func (r *Repository) Set(ctx context.Context, ent entity.Entity, expiration time.Duration) error {
	if err := r.ValidateEntity(ent); err != nil {
		return err
	}

	id := ent.GetID()
	key := r.keyPrefix + id

	// Set timestamps
	now := time.Now()
	if ent.GetCreatedAt().IsZero() {
		ent.SetCreatedAt(now)
	}
	ent.SetUpdatedAt(now)

	err := r.service.SetJSON(ctx, key, ent, expiration)
	if err != nil {
		return r.HandleUpdateError(err, "set", id)
	}

	return nil
}

// SetWithTTL stores an entity with time-to-live.
func (r *Repository) SetWithTTL(ctx context.Context, ent entity.Entity, ttl time.Duration) error {
	return r.Set(ctx, ent, ttl)
}

// GetWithTTL retrieves an entity and its remaining TTL.
func (r *Repository) GetWithTTL(ctx context.Context, id string) (entity.Entity, time.Duration, error) {
	if err := r.ValidateID(id); err != nil {
		return nil, 0, err
	}

	key := r.keyPrefix + id

	// Get entity
	ent, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, 0, err
	}

	// Get TTL
	ttl, err := r.service.TTL(ctx, key)
	if err != nil {
		return ent, -1, nil // Return entity even if TTL query fails
	}

	return ent, ttl, nil
}

// List retrieves entities with pattern-based pagination.
func (r *Repository) List(ctx context.Context, pageSize int32, pageToken string) ([]entity.Entity, string, error) {
	pattern := r.keyPrefix + "*"

	keys, nextToken, err := r.service.ScanWithPagination(ctx, pattern, pageSize, pageToken)
	if err != nil {
		return nil, "", r.HandleGetError(err, "list", "")
	}

	// Get all entities
	var entities []entity.Entity
	if len(keys) > 0 {
		values, err := r.service.MGet(ctx, keys)
		if err != nil {
			return nil, "", r.HandleBatchError(err, "mget", []any{keys})
		}

		for _, key := range keys {
			if data, exists := values[key]; exists {
				newEntity := r.CreateNewEntity()
				if err := json.Unmarshal(data, newEntity); err == nil {
					entities = append(entities, newEntity)
				}
			}
		}
	}

	return entities, nextToken, nil
}

// ListByPattern retrieves entities matching a specific pattern.
func (r *Repository) ListByPattern(ctx context.Context, pattern string, pageSize int32, pageToken string) ([]entity.Entity, string, error) {
	fullPattern := r.keyPrefix + pattern

	keys, nextToken, err := r.service.ScanWithPagination(ctx, fullPattern, pageSize, pageToken)
	if err != nil {
		return nil, "", r.HandleGetError(err, "list_pattern", "")
	}

	// Get all entities
	var entities []entity.Entity
	if len(keys) > 0 {
		values, err := r.service.MGet(ctx, keys)
		if err != nil {
			return nil, "", r.HandleBatchError(err, "mget_pattern", []any{keys})
		}

		for _, key := range keys {
			if data, exists := values[key]; exists {
				newEntity := r.CreateNewEntity()
				if err := json.Unmarshal(data, newEntity); err == nil {
					entities = append(entities, newEntity)
				}
			}
		}
	}

	return entities, nextToken, nil
}

// Batch operations

// SetBatch stores multiple entities.
func (r *Repository) SetBatch(ctx context.Context, entities []entity.Entity, expiration time.Duration) error {
	if len(entities) == 0 {
		return nil
	}

	// Validate all entities first
	for _, ent := range entities {
		if err := r.ValidateEntity(ent); err != nil {
			return err
		}
	}

	pairs := make(map[string][]byte)
	now := time.Now()

	for _, ent := range entities {
		id := ent.GetID()

		// Set timestamps
		if ent.GetCreatedAt().IsZero() {
			ent.SetCreatedAt(now)
		}
		ent.SetUpdatedAt(now)

		key := r.keyPrefix + id
		data, err := json.Marshal(ent)
		if err != nil {
			return fmt.Errorf("failed to marshal entity %s: %w", id, err)
		}

		pairs[key] = data
	}

	err := r.service.MSet(ctx, pairs, expiration)
	if err != nil {
		return r.HandleBatchError(err, "set_batch", []any{entities})
	}

	return nil
}

// GetBatch retrieves multiple entities by IDs.
func (r *Repository) GetBatch(ctx context.Context, ids []string) (map[string]entity.Entity, error) {
	if len(ids) == 0 {
		return make(map[string]entity.Entity), nil
	}

	// Validate all IDs first
	for _, id := range ids {
		if err := r.ValidateID(id); err != nil {
			return nil, err
		}
	}

	// Build keys
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = r.keyPrefix + id
	}

	// Get data
	values, err := r.service.MGet(ctx, keys)
	if err != nil {
		return nil, r.HandleBatchError(err, "get_batch", []any{ids})
	}

	// Unmarshal entities
	result := make(map[string]entity.Entity)
	for i, key := range keys {
		if data, exists := values[key]; exists {
			newEntity := r.CreateNewEntity()
			if err := json.Unmarshal(data, newEntity); err == nil {
				result[ids[i]] = newEntity
			}
		}
	}

	return result, nil
}

// DeleteBatch deletes multiple entities by IDs.
func (r *Repository) DeleteBatch(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Validate all IDs first
	for _, id := range ids {
		if err := r.ValidateID(id); err != nil {
			return err
		}
	}

	// Build keys
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = r.keyPrefix + id
	}

	err := r.service.MDelete(ctx, keys)
	if err != nil {
		return r.HandleBatchError(err, "delete_batch", []any{ids})
	}

	return nil
}

// Expiration operations

// SetExpiration sets expiration for an entity.
func (r *Repository) SetExpiration(ctx context.Context, id string, expiration time.Duration) error {
	if err := r.ValidateID(id); err != nil {
		return err
	}

	key := r.keyPrefix + id

	err := r.service.Expire(ctx, key, expiration)
	if err != nil {
		return r.HandleUpdateError(err, "expire", id)
	}

	return nil
}

// GetTTL returns time-to-live for an entity.
func (r *Repository) GetTTL(ctx context.Context, id string) (time.Duration, error) {
	if err := r.ValidateID(id); err != nil {
		return 0, err
	}

	key := r.keyPrefix + id

	ttl, err := r.service.TTL(ctx, key)
	if err != nil {
		return 0, r.HandleGetError(err, "ttl", id)
	}

	return ttl, nil
}

// Atomic operations (if supported by adapter)

// IncrementField increments a numeric field in an entity (simplified implementation).
func (r *Repository) IncrementField(ctx context.Context, id string, field string, value int64) (int64, error) {
	// This would need more sophisticated implementation for real atomic operations
	// For now, return an error indicating limited support
	return 0, fmt.Errorf("atomic field operations not supported in KV repository")
}

// Accessors

// Service returns the underlying KV service.
func (r *Repository) Service() *Service {
	return r.service
}

// KeyPrefix returns the key prefix used for this entity type.
func (r *Repository) KeyPrefix() string {
	return r.keyPrefix
}
