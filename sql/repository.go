package sqlstore

import (
	"context"
	"database/sql"
	"time"

	"core/entity"
	"store"
)

// Repository provides SQL storage for a specific entity type.
// It leverages core/entity for reflection and scanning, and extends the shared base.
type Repository struct {
	*store.RepositoryBase
	service   *Service
	tableName string

	transactionHandler *TransactionHandler
	queryExecutor      *QueryExecutor
	paginator          *SQLPaginator
}

// Ensure Repository satisfies store-agnostic contracts.
var _ store.EntityRepository[entity.Entity] = (*Repository)(nil)
var _ store.Countable = (*Repository)(nil)

// NewRepository creates a new entity-specific repository.
func NewRepository(service *Service, ent entity.Entity) *Repository {
	tableName := entity.GetTableName(ent)

	return &Repository{
		RepositoryBase:     store.NewRepositoryBase(ent),
		service:            service,
		tableName:          tableName,
		transactionHandler: NewTransactionHandler(service.db, service.adapter),
		queryExecutor:      NewQueryExecutor(service.db),
		paginator:          NewSQLPaginator(),
	}
}

// Entity-agnostic interface implementation

// GetByID retrieves an entity by ID (tech-agnostic signature).
func (r *Repository) GetByID(ctx context.Context, id string) (entity.Entity, error) {
	if err := r.ValidateID(id); err != nil {
		return nil, err
	}
	return r.GetByIDWithColumns(ctx, id)
}

// Exists checks if an entity exists by ID.
func (r *Repository) Exists(ctx context.Context, id string) (bool, error) {
	if err := r.ValidateID(id); err != nil {
		return false, err
	}

	qb := r.Find().WhereEq("id", id)
	exists, err := r.queryExecutor.Exists(ctx, qb)
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

	db := r.Delete().WhereEq("id", id)
	res, err := r.queryExecutor.ExecuteDelete(ctx, db)
	if err != nil {
		return r.HandleUpdateError(err, "delete", id)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return r.HandleUpdateError(err, "rows_affected", id)
	}

	if rows == 0 {
		return store.NewRecordNotFoundError(r.EntityName(), id)
	}

	return nil
}

// Count returns the total count of entities.
func (r *Repository) Count(ctx context.Context) (int64, error) {
	qb := r.Find()
	count, err := r.queryExecutor.Count(ctx, qb)
	if err != nil {
		return 0, r.HandleGetError(err, "count", "")
	}
	return count, nil
}

// SQL-specific methods with enhanced functionality

// GetByIDWithColumns retrieves an entity by ID with optional column projection.
func (r *Repository) GetByIDWithColumns(ctx context.Context, id string, columns ...string) (entity.Entity, error) {
	var result entity.Entity

	err := r.transactionHandler.WithReadTx(ctx, func(ctxTx context.Context) error {
		qb := r.Find()
		if len(columns) > 0 {
			qb.Select(columns...)
		}
		qb.WhereEq("id", id)

		row := r.queryExecutor.QueryRow(ctxTx, qb)
		ent, err := r.scanRow(row)
		if err != nil {
			if err == sql.ErrNoRows {
				return store.NewRecordNotFoundError(r.EntityName(), id)
			}
			return err
		}

		result = ent
		return nil
	})

	return result, err
}

// List retrieves entities with pagination using the new pagination types.
func (r *Repository) List(ctx context.Context, pageSize int32, pageToken string, columns ...string) ([]entity.Entity, string, error) {
	var entities []entity.Entity
	var nextToken string

	err := r.transactionHandler.WithReadTx(ctx, func(ctxTx context.Context) error {
		params := r.paginator.ParseParams(pageSize, pageToken)
		qb := r.Find()
		if len(columns) > 0 {
			qb.Select(columns...)
		}

		result, err := ExecutePaginatedQuery(ctxTx, r.paginator, r.queryExecutor, qb, params, r.scanRowsEntity)
		if err != nil {
			return err
		}

		entities = result.Items
		nextToken = result.NextCursor

		return nil
	})

	return entities, nextToken, err
}

// UpdateTimestamp updates the updated_at field for an entity.
func (r *Repository) UpdateTimestamp(ctx context.Context, id string) error {
	if err := r.ValidateID(id); err != nil {
		return err
	}

	ub := r.Update().
		Set("updated_at", time.Now()).
		WhereEq("id", id)

	_, err := r.queryExecutor.ExecuteUpdate(ctx, ub)
	return r.HandleUpdateError(err, "update_timestamp", id)
}

// Query builders

// NewQueryBuilder creates a new query builder for this entity's table.
func (r *Repository) Find() *QueryBuilder { return NewQueryBuilder(r.tableName) }

// NewUpdateBuilder creates a new update builder for this entity's table.
func (r *Repository) Update() *UpdateBuilder { return NewUpdateBuilder(r.tableName) }

// NewDeleteBuilder creates a new delete builder for this entity's table.
func (r *Repository) Delete() *DeleteBuilder { return NewDeleteBuilder(r.tableName) }

// ExecutePaginatedQuery executes a paginated query with custom scanning (legacy support).
func (r *Repository) ExecutePaginatedQuery(ctx context.Context, qb *QueryBuilder, pageSize int32, pageToken string, scanFunc func(*sql.Rows) (any, error)) (*PaginationResult, error) {
	params := r.paginator.ParseParams(pageSize, pageToken)
	return r.paginator.ExecutePaginatedQueryWithScan(ctx, r.queryExecutor, qb, params, scanFunc)
}

// Scanning helpers using core/entity

// scanRow scans a single row into a new entity instance.
func (r *Repository) scanRow(row *sql.Row) (entity.Entity, error) {
	newEntity := r.CreateNewEntity()
	if err := entity.ScanEntity(newEntity, row); err != nil {
		return nil, err
	}
	return newEntity, nil
}

// scanRowsEntity scans sql.Rows into an entity (for new pagination).
func (r *Repository) scanRowsEntity(rows *sql.Rows) (entity.Entity, error) {
	newEntity := r.CreateNewEntity()
	if err := r.scanEntityFromRows(newEntity, rows); err != nil {
		return nil, err
	}
	return newEntity, nil
}

// scanRows scans multiple rows (for legacy pagination).
func (r *Repository) scanRows(rows *sql.Rows) (any, error) { return r.scanRowsEntity(rows) }

// scanEntityFromRows adapts core/entity scanning for sql.Rows.
func (r *Repository) scanEntityFromRows(ent entity.Entity, rows *sql.Rows) error {
	// This would need to be implemented to work with core/entity
	// For now, use a simplified approach
	return r.scanIntoStruct(ent, rows)
}

// scanIntoStruct provides basic struct scanning using db tags.
func (r *Repository) scanIntoStruct(target entity.Entity, scanner interface{}) error {
	if row, ok := scanner.(*sql.Row); ok {
		return entity.ScanEntity(target, row)
	}
	return entity.ScanEntity(target, scanner.(*sql.Row))
}

// Accessors

// Service returns the underlying SQL service.
func (r *Repository) Service() *Service { return r.service }

// TableName returns the entity's table name.
func (r *Repository) TableName() string { return r.tableName }
