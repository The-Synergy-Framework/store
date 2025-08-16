package store

import (
	"context"
	"time"

	"core/entity"
	"core/validation"
)

// RepositoryBase provides common functionality for all repository implementations.
type RepositoryBase struct {
	entityName     string
	tableName      string
	newEntityFunc  func() entity.Entity
	validator      validation.Validator
	metricsEnabled bool
}

// NewRepositoryBase creates a new base repository.
func NewRepositoryBase(ent entity.Entity) *RepositoryBase {
	return &RepositoryBase{
		entityName:     entity.GetEntityName(ent),
		tableName:      entity.GetTableName(ent),
		newEntityFunc:  func() entity.Entity { return entity.CreateNewEntity(ent) },
		validator:      nil, // Use default validation.Validate function
		metricsEnabled: true,
	}
}

// EntityName returns the entity name.
func (r *RepositoryBase) EntityName() string {
	return r.entityName
}

// TableName returns the table name.
func (r *RepositoryBase) TableName() string {
	return r.tableName
}

// CreateNewEntity creates a new entity instance.
func (r *RepositoryBase) CreateNewEntity() entity.Entity {
	return r.newEntityFunc()
}

// Validate validates an entity.
func (r *RepositoryBase) Validate(ctx context.Context, ent entity.Entity) error {
	// Use the default validation function
	result := validation.Validate(ent)
	if !result.IsValid {
		return NewValidationErrorFromResult(result, ent)
	}

	return nil
}

// ValidateID validates an entity ID.
func (r *RepositoryBase) ValidateID(id string) error {
	if id == "" {
		return NewValidationError("entity ID cannot be empty")
	}
	return nil
}

// SetTimestamps sets created_at and updated_at timestamps.
func (r *RepositoryBase) SetTimestamps(ent entity.Entity, isCreate bool) {
	now := time.Now()
	if isCreate {
		ent.SetCreatedAt(now)
	}
	ent.SetUpdatedAt(now)
}

// Error handling helpers

// HandleGetError wraps get operation errors with context.
func (r *RepositoryBase) HandleGetError(err error, operation, id string) error {
	if err == nil {
		return nil
	}
	return WrapRepositoryError(err, r.entityName, operation, map[string]any{"id": id})
}

// HandleUpdateError wraps update operation errors with context.
func (r *RepositoryBase) HandleUpdateError(err error, operation, id string) error {
	if err == nil {
		return nil
	}
	return WrapRepositoryError(err, r.entityName, operation, map[string]any{"id": id})
}

// HandleQueryError wraps query operation errors with context.
func (r *RepositoryBase) HandleQueryError(err error, operation string, context map[string]any) error {
	if err == nil {
		return nil
	}
	return WrapRepositoryError(err, r.entityName, operation, context)
}
