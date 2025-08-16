package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type QueryBuilder struct {
	table    string
	columns  []string
	where    []Condition
	orderBy  []OrderBy
	limit    *int
	offset   *int
	args     []interface{}
	argIndex int
}

type Condition struct {
	Column    string
	Operator  string
	Value     interface{}
	Connector string
}

type OrderBy struct {
	Column    string
	Direction string
}

func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{table: table, columns: []string{"*"}, where: []Condition{}, orderBy: []OrderBy{}, args: []interface{}{}, argIndex: 1}
}

func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	if len(columns) > 0 {
		qb.columns = columns
	}
	return qb
}

func (qb *QueryBuilder) Where(column, operator string, value interface{}) *QueryBuilder {
	if value == nil {
		return qb
	}
	if s, ok := value.(string); ok && s == "" {
		return qb
	}
	qb.where = append(qb.where, Condition{Column: column, Operator: operator, Value: value, Connector: "AND"})
	return qb
}

func (qb *QueryBuilder) WhereEq(column string, value interface{}) *QueryBuilder {
	return qb.Where(column, "=", value)
}

func (qb *QueryBuilder) WhereNull(column string) *QueryBuilder {
	qb.where = append(qb.where, Condition{Column: fmt.Sprintf("%s IS NULL", column), Connector: "AND"})
	return qb
}

func (qb *QueryBuilder) WhereNotNull(column string) *QueryBuilder {
	qb.where = append(qb.where, Condition{Column: fmt.Sprintf("%s IS NOT NULL", column), Connector: "AND"})
	return qb
}

func (qb *QueryBuilder) OrderBy(column, direction string) *QueryBuilder {
	qb.orderBy = append(qb.orderBy, OrderBy{Column: column, Direction: strings.ToUpper(direction)})
	return qb
}
func (qb *QueryBuilder) OrderByAsc(column string) *QueryBuilder  { return qb.OrderBy(column, "ASC") }
func (qb *QueryBuilder) OrderByDesc(column string) *QueryBuilder { return qb.OrderBy(column, "DESC") }
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder           { qb.limit = &limit; return qb }
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder         { qb.offset = &offset; return qb }
func (qb *QueryBuilder) Paginate(pageSize, offset int) *QueryBuilder {
	return qb.Limit(pageSize).Offset(offset)
}

func (qb *QueryBuilder) buildWhereClause() string {
	var parts []string
	for i, c := range qb.where {
		var s string
		if c.Operator == "" {
			s = c.Column
		} else {
			s = fmt.Sprintf("%s %s $%d", c.Column, c.Operator, qb.argIndex)
			qb.args = append(qb.args, c.Value)
			qb.argIndex++
		}
		if i > 0 {
			parts = append(parts, c.Connector)
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}

func (qb *QueryBuilder) buildOrderByClause() string {
	var parts []string
	for _, ob := range qb.orderBy {
		parts = append(parts, fmt.Sprintf("%s %s", ob.Column, ob.Direction))
	}
	return strings.Join(parts, ", ")
}

func (qb *QueryBuilder) Build() (string, []interface{}) {
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(qb.columns, ", "), qb.table)
	if len(qb.where) > 0 {
		query += " WHERE " + qb.buildWhereClause()
	}
	if len(qb.orderBy) > 0 {
		query += " ORDER BY " + qb.buildOrderByClause()
	}
	if qb.limit != nil {
		query += fmt.Sprintf(" LIMIT $%d", qb.argIndex)
		qb.args = append(qb.args, *qb.limit)
		qb.argIndex++
	}
	if qb.offset != nil {
		query += fmt.Sprintf(" OFFSET $%d", qb.argIndex)
		qb.args = append(qb.args, *qb.offset)
	}
	return query, qb.args
}

// Update

type UpdateBuilder struct {
	table    string
	updates  map[string]interface{}
	where    []Condition
	args     []interface{}
	argIndex int
}

func NewUpdateBuilder(table string) *UpdateBuilder {
	return &UpdateBuilder{table: table, updates: map[string]interface{}{}, where: []Condition{}, args: []interface{}{}, argIndex: 1}
}
func (ub *UpdateBuilder) Set(column string, value interface{}) *UpdateBuilder {
	ub.updates[column] = value
	return ub
}
func (ub *UpdateBuilder) SetTimestamp() *UpdateBuilder { return ub.Set("updated_at", time.Now()) }
func (ub *UpdateBuilder) Where(column, operator string, value interface{}) *UpdateBuilder {
	if value == nil {
		return ub
	}
	if s, ok := value.(string); ok && s == "" {
		return ub
	}
	ub.where = append(ub.where, Condition{Column: column, Operator: operator, Value: value, Connector: "AND"})
	return ub
}
func (ub *UpdateBuilder) WhereEq(column string, value interface{}) *UpdateBuilder {
	return ub.Where(column, "=", value)
}
func (ub *UpdateBuilder) buildWhereClause() string {
	var parts []string
	for i, c := range ub.where {
		var s string
		if c.Operator == "" {
			s = c.Column
		} else {
			s = fmt.Sprintf("%s %s $%d", c.Column, c.Operator, ub.argIndex)
			ub.args = append(ub.args, c.Value)
			ub.argIndex++
		}
		if i > 0 {
			parts = append(parts, c.Connector)
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}
func (ub *UpdateBuilder) Build() (string, []interface{}) {
	sets := make([]string, 0, len(ub.updates))
	for col, v := range ub.updates {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, ub.argIndex))
		ub.args = append(ub.args, v)
		ub.argIndex++
	}
	q := fmt.Sprintf("UPDATE %s SET %s", ub.table, strings.Join(sets, ", "))
	if len(ub.where) > 0 {
		q += " WHERE " + ub.buildWhereClause()
	}
	return q, ub.args
}

// Delete

type DeleteBuilder struct {
	table    string
	where    []Condition
	args     []interface{}
	argIndex int
}

func NewDeleteBuilder(table string) *DeleteBuilder {
	return &DeleteBuilder{table: table, where: []Condition{}, args: []interface{}{}, argIndex: 1}
}
func (db *DeleteBuilder) Where(column, operator string, value interface{}) *DeleteBuilder {
	if value == nil {
		return db
	}
	if s, ok := value.(string); ok && s == "" {
		return db
	}
	db.where = append(db.where, Condition{Column: column, Operator: operator, Value: value, Connector: "AND"})
	return db
}
func (db *DeleteBuilder) WhereEq(column string, value interface{}) *DeleteBuilder {
	return db.Where(column, "=", value)
}
func (db *DeleteBuilder) buildWhereClause() string {
	var parts []string
	for i, c := range db.where {
		var s string
		if c.Operator == "" {
			s = c.Column
		} else {
			s = fmt.Sprintf("%s %s $%d", c.Column, c.Operator, db.argIndex)
			db.args = append(db.args, c.Value)
			db.argIndex++
		}
		if i > 0 {
			parts = append(parts, c.Connector)
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}
func (db *DeleteBuilder) Build() (string, []interface{}) {
	q := fmt.Sprintf("DELETE FROM %s", db.table)
	if len(db.where) > 0 {
		q += " WHERE " + db.buildWhereClause()
	}
	return q, db.args
}

// Executor

type QueryExecutor struct{ db *sql.DB }

func NewQueryExecutor(db *sql.DB) *QueryExecutor { return &QueryExecutor{db: db} }
func (qe *QueryExecutor) Query(ctx context.Context, qb *QueryBuilder) (*sql.Rows, error) {
	q, a := qb.Build()
	if tx, ok := TransactionFromContext(ctx); ok && tx != nil {
		return tx.QueryContext(ctx, q, a...)
	}
	return qe.db.QueryContext(ctx, q, a...)
}
func (qe *QueryExecutor) QueryRow(ctx context.Context, qb *QueryBuilder) *sql.Row {
	q, a := qb.Build()
	if tx, ok := TransactionFromContext(ctx); ok && tx != nil {
		return tx.QueryRowContext(ctx, q, a...)
	}
	return qe.db.QueryRowContext(ctx, q, a...)
}
func (qe *QueryExecutor) Count(ctx context.Context, qb *QueryBuilder) (int64, error) {
	cq := NewQueryBuilder(qb.table).Select("COUNT(*)")
	cq.where = append(cq.where, qb.where...)
	var count int64
	q, a := cq.Build()
	if tx, ok := TransactionFromContext(ctx); ok && tx != nil {
		err := tx.QueryRowContext(ctx, q, a...).Scan(&count)
		return count, err
	}
	err := qe.db.QueryRowContext(ctx, q, a...).Scan(&count)
	return count, err
}
func (qe *QueryExecutor) Exists(ctx context.Context, qb *QueryBuilder) (bool, error) {
	exq := NewQueryBuilder(qb.table).Select("1")
	exq.where = append(exq.where, qb.where...)
	exq.Limit(1)
	q, a := exq.Build()
	var one int
	if tx, ok := TransactionFromContext(ctx); ok && tx != nil {
		err := tx.QueryRowContext(ctx, q, a...).Scan(&one)
		if err == sql.ErrNoRows {
			return false, nil
		}
		return err == nil, err
	}
	err := qe.db.QueryRowContext(ctx, q, a...).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
func (qe *QueryExecutor) ExecuteUpdate(ctx context.Context, ub *UpdateBuilder) (sql.Result, error) {
	q, a := ub.Build()
	if tx, ok := TransactionFromContext(ctx); ok && tx != nil {
		return tx.ExecContext(ctx, q, a...)
	}
	return qe.db.ExecContext(ctx, q, a...)
}
func (qe *QueryExecutor) ExecuteDelete(ctx context.Context, db *DeleteBuilder) (sql.Result, error) {
	q, a := db.Build()
	if tx, ok := TransactionFromContext(ctx); ok && tx != nil {
		return tx.ExecContext(ctx, q, a...)
	}
	return qe.db.ExecContext(ctx, q, a...)
}

// ExecuteCompiled provides execution for compiled SQL.
func (qe *QueryExecutor) ExecuteCompiled(ctx context.Context, c *CompiledSQL) (*sql.Rows, error) {
	if tx, ok := TransactionFromContext(ctx); ok && tx != nil {
		return tx.QueryContext(ctx, c.SQL, c.Args...)
	}
	return qe.db.QueryContext(ctx, c.SQL, c.Args...)
}

// ExecuteCompiledRow runs a compiled SQL expected to return a single row.
func (qe *QueryExecutor) ExecuteCompiledRow(ctx context.Context, c *CompiledSQL) *sql.Row {
	if tx, ok := TransactionFromContext(ctx); ok && tx != nil {
		return tx.QueryRowContext(ctx, c.SQL, c.Args...)
	}
	return qe.db.QueryRowContext(ctx, c.SQL, c.Args...)
}

// ExecuteCompiledExec runs a compiled SQL that doesn't return rows.
func (qe *QueryExecutor) ExecuteCompiledExec(ctx context.Context, c *CompiledSQL) (sql.Result, error) {
	if tx, ok := TransactionFromContext(ctx); ok && tx != nil {
		return tx.ExecContext(ctx, c.SQL, c.Args...)
	}
	return qe.db.ExecContext(ctx, c.SQL, c.Args...)
}
