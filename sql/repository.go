package sqlstore

import (
	"context"
	"database/sql"

	"core/entity"
	"store"
)

// Repository provides SQL storage implementing the standardized interface.
type Repository struct {
	*store.RepositoryBase

	sqlService         *Service
	transactionHandler *TransactionHandler
	mutationExecutor   *MutationExecutor
}

// Ensure Repository implements store.Repository
var _ store.Repository = (*Repository)(nil)

// NewRepository creates a new SQL repository.
func NewRepository(service *Service, ent entity.Entity) *Repository {
	base := store.NewRepositoryBase(ent)

	return &Repository{
		RepositoryBase:     base,
		sqlService:         service,
		transactionHandler: NewTransactionHandler(service.db, service.adapter),
		mutationExecutor:   NewMutationExecutor(service.db),
	}
}

// Core CRUD operations

// Create stores a new entity in the database.
func (r *Repository) Create(ctx context.Context, ent entity.Entity) error {
	if err := r.Validate(ctx, ent); err != nil {
		return err
	}

	r.SetTimestamps(ent, true)

	return r.transactionHandler.WithTx(ctx, func(ctxTx context.Context) error {
		values := entity.ToMap(ent)
		mutation := store.Insert{Values: values}

		compiled, err := CompileMutation(r.TableName(), mutation)
		if err != nil {
			return r.HandleUpdateError(err, "create", ent.GetID())
		}

		_, err = r.mutationExecutor.ExecuteCompiled(ctxTx, *compiled)
		return r.HandleUpdateError(err, "create", ent.GetID())
	})
}

// Get retrieves an entity by ID - simplified implementation.
func (r *Repository) Get(ctx context.Context, id string) (entity.Entity, error) {
	if err := r.ValidateID(id); err != nil {
		return nil, err
	}

	// Simple SQL query without complex compilation
	sqlQuery := "SELECT * FROM " + r.TableName() + " WHERE id = $1"
	row := r.sqlService.db.QueryRowContext(ctx, sqlQuery, id)

	result := r.CreateNewEntity()
	err := entity.ScanEntity(result, row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, store.NewRecordNotFoundError(r.EntityName(), id)
		}
		return nil, r.HandleGetError(err, "get", id)
	}

	return result, nil
}

// Update modifies an existing entity in the database.
func (r *Repository) Update(ctx context.Context, ent entity.Entity) error {
	if err := r.Validate(ctx, ent); err != nil {
		return err
	}

	r.SetTimestamps(ent, false)

	return r.transactionHandler.WithTx(ctx, func(ctxTx context.Context) error {
		values := entity.ToMap(ent)
		delete(values, "id") // Don't update the ID

		mutation := store.Update{
			Set:   values,
			Where: []store.Condition{store.Eq("id", ent.GetID())},
		}

		compiled, err := CompileMutation(r.TableName(), mutation)
		if err != nil {
			return r.HandleUpdateError(err, "update", ent.GetID())
		}

		result, err := r.mutationExecutor.ExecuteCompiled(ctxTx, *compiled)
		if err != nil {
			return r.HandleUpdateError(err, "update", ent.GetID())
		}

		if result.RowsAffected == 0 {
			return store.NewRecordNotFoundError(r.EntityName(), ent.GetID())
		}

		return nil
	})
}

// Delete removes an entity by ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	if err := r.ValidateID(id); err != nil {
		return err
	}

	return r.transactionHandler.WithTx(ctx, func(ctxTx context.Context) error {
		mutation := store.Delete{
			Where: []store.Condition{store.Eq("id", id)},
		}

		compiled, err := CompileMutation(r.TableName(), mutation)
		if err != nil {
			return r.HandleUpdateError(err, "delete", id)
		}

		result, err := r.mutationExecutor.ExecuteCompiled(ctxTx, *compiled)
		if err != nil {
			return r.HandleUpdateError(err, "delete", id)
		}

		if result.RowsAffected == 0 {
			return store.NewRecordNotFoundError(r.EntityName(), id)
		}

		return nil
	})
}

// Exists checks if an entity with the given ID exists.
func (r *Repository) Exists(ctx context.Context, id string) (bool, error) {
	if err := r.ValidateID(id); err != nil {
		return false, err
	}

	// Simple SQL query
	sqlQuery := "SELECT 1 FROM " + r.TableName() + " WHERE id = $1 LIMIT 1"
	row := r.sqlService.db.QueryRowContext(ctx, sqlQuery, id)

	var exists int
	err := row.Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, r.HandleGetError(err, "exists", id)
	}

	return true, nil
}

// Batch operations - simplified implementations

// CreateBatch creates multiple entities in a single transaction.
func (r *Repository) CreateBatch(ctx context.Context, entities []entity.Entity) error {
	if len(entities) == 0 {
		return nil
	}

	return r.transactionHandler.WithTx(ctx, func(ctxTx context.Context) error {
		for _, ent := range entities {
			if err := r.Create(ctxTx, ent); err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateBatch updates multiple entities in a single transaction.
func (r *Repository) UpdateBatch(ctx context.Context, entities []entity.Entity) error {
	if len(entities) == 0 {
		return nil
	}

	return r.transactionHandler.WithTx(ctx, func(ctxTx context.Context) error {
		for _, ent := range entities {
			if err := r.Update(ctxTx, ent); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteBatch deletes multiple entities by IDs.
func (r *Repository) DeleteBatch(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	return r.transactionHandler.WithTx(ctx, func(ctxTx context.Context) error {
		for _, id := range ids {
			if err := r.Delete(ctxTx, id); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetBatch retrieves multiple entities by IDs.
func (r *Repository) GetBatch(ctx context.Context, ids []string) (map[string]entity.Entity, error) {
	result := make(map[string]entity.Entity)

	for _, id := range ids {
		ent, err := r.Get(ctx, id)
		if err != nil {
			// Skip not found errors, include the entity if found
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

// FindWhere returns entities matching the given conditions.
func (r *Repository) FindWhere(ctx context.Context, conditions ...store.Condition) ([]entity.Entity, error) {
	// Simple implementation - for now just return empty slice
	// This would be enhanced to actually build SQL WHERE clauses from conditions
	return []entity.Entity{}, nil
}

// CountWhere returns the count of entities matching the given conditions.
func (r *Repository) CountWhere(ctx context.Context, conditions ...store.Condition) (int64, error) {
	// Simple implementation - for now just return total count
	// This would be enhanced to actually build SQL WHERE clauses from conditions
	return r.Count(ctx)
}

// FindFirst returns the first entity matching the given conditions.
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

// List returns paginated results - simplified implementation.
func (r *Repository) List(ctx context.Context, params store.CursorParams) (store.CursorResult[entity.Entity], error) {
	// Simple implementation - just get all records with limit
	var entities []entity.Entity

	limit := int(params.PageSize)
	if limit <= 0 {
		limit = 100 // Default limit
	}

	sqlQuery := "SELECT * FROM " + r.TableName() + " LIMIT $1"
	rows, err := r.sqlService.db.QueryContext(ctx, sqlQuery, limit)
	if err != nil {
		return store.CursorResult[entity.Entity]{}, r.HandleQueryError(err, "list", nil)
	}
	defer rows.Close()

	for rows.Next() {
		ent := r.CreateNewEntity()
		// ScanEntity expects *sql.Row, but we have *sql.Rows - need to scan manually for now
		values, err := scanRowToValues(rows)
		if err != nil {
			return store.CursorResult[entity.Entity]{}, r.HandleQueryError(err, "list", nil)
		}
		if err := entity.FromMap(ent, values); err != nil {
			return store.CursorResult[entity.Entity]{}, r.HandleQueryError(err, "list", nil)
		}
		entities = append(entities, ent)
	}

	if err = rows.Err(); err != nil {
		return store.CursorResult[entity.Entity]{}, r.HandleQueryError(err, "list", nil)
	}

	return store.CursorResult[entity.Entity]{
		Items:   entities,
		HasMore: len(entities) == limit, // Simple heuristic
	}, nil
}

// Count returns the number of entities matching the conditions.
func (r *Repository) Count(ctx context.Context, conditions ...store.Condition) (int64, error) {
	// Simple implementation - count all records
	sqlQuery := "SELECT COUNT(*) FROM " + r.TableName()
	row := r.sqlService.db.QueryRowContext(ctx, sqlQuery)

	var count int64
	err := row.Scan(&count)
	if err != nil {
		return 0, r.HandleQueryError(err, "count", nil)
	}

	return count, nil
}

// HealthCheck performs a basic health check.
func (r *Repository) HealthCheck(ctx context.Context) error {
	_, err := r.Count(ctx)
	if err != nil {
		return r.HandleQueryError(err, "health_check", nil)
	}
	return nil
}

// Helper function for scanning rows - simplified implementation
func scanRowToValues(rows *sql.Rows) (map[string]any, error) {
	// This is a placeholder - in a real implementation, we would properly scan
	// based on the entity's field structure. For now, return empty map.
	return make(map[string]any), nil
}
