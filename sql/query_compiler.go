package sqlstore

import (
	"fmt"
	"strings"

	"store"
)

// CompiledSQL represents a compiled SQL statement with arguments.
type CompiledSQL struct {
	SQL  string
	Args []any
}

// SQLCompiler compiles a backend-agnostic store.Query into SQL.
type SQLCompiler struct {
	table string
}

func NewSQLCompiler(table string) *SQLCompiler { return &SQLCompiler{table: table} }

func (c *SQLCompiler) Compile(q store.Query) (*CompiledSQL, error) {
	qb := NewQueryBuilder(c.table)
	if len(q.SelectFields) > 0 {
		qb.Select(q.SelectFields...)
	}

	args := []any{}
	argIndex := 1

	// where clause
	if q.Filter != nil {
		wsql, wargs := c.compileNode(q.Filter, &argIndex)
		if wsql != "" {
			qb.where = append(qb.where, Condition{Column: wsql})
			args = append(args, wargs...)
		}
	}

	// order by
	for _, o := range q.OrderBy {
		if o.Desc {
			qb.OrderByDesc(o.Field)
		} else {
			qb.OrderByAsc(o.Field)
		}
	}

	if q.Limit != nil {
		qb.Limit(*q.Limit)
	}
	if q.Offset != nil {
		qb.Offset(*q.Offset)
	}

	sql, _ := qb.Build()
	// Replace builder's internally collected args with our flattened args
	return &CompiledSQL{SQL: sql, Args: args}, nil
}

func (c *SQLCompiler) compileNode(n store.Node, argIndex *int) (string, []any) {
	switch v := n.(type) {
	case store.Condition:
		return c.compileCondition(v, argIndex)
	case store.And:
		parts := make([]string, 0, len(v.Children))
		var args []any
		for _, ch := range v.Children {
			s, a := c.compileNode(ch, argIndex)
			if s != "" {
				parts = append(parts, s)
				args = append(args, a...)
			}
		}
		if len(parts) == 0 {
			return "", nil
		}
		return "(" + strings.Join(parts, " AND ") + ")", args
	case store.Or:
		parts := make([]string, 0, len(v.Children))
		var args []any
		for _, ch := range v.Children {
			s, a := c.compileNode(ch, argIndex)
			if s != "" {
				parts = append(parts, s)
				args = append(args, a...)
			}
		}
		if len(parts) == 0 {
			return "", nil
		}
		return "(" + strings.Join(parts, " OR ") + ")", args
	default:
		return "", nil
	}
}

func (c *SQLCompiler) compileCondition(cond store.Condition, argIndex *int) (string, []any) {
	f := cond.Field
	switch cond.Op {
	case store.OpEq:
		s := fmt.Sprintf("%s = $%d", f, *argIndex)
		*argIndex++
		return s, []any{cond.Value}
	case store.OpNe:
		s := fmt.Sprintf("%s <> $%d", f, *argIndex)
		*argIndex++
		return s, []any{cond.Value}
	case store.OpGt:
		s := fmt.Sprintf("%s > $%d", f, *argIndex)
		*argIndex++
		return s, []any{cond.Value}
	case store.OpGe:
		s := fmt.Sprintf("%s >= $%d", f, *argIndex)
		*argIndex++
		return s, []any{cond.Value}
	case store.OpLt:
		s := fmt.Sprintf("%s < $%d", f, *argIndex)
		*argIndex++
		return s, []any{cond.Value}
	case store.OpLe:
		s := fmt.Sprintf("%s <= $%d", f, *argIndex)
		*argIndex++
		return s, []any{cond.Value}
	case store.OpIn:
		vals, _ := cond.Value.([]any)
		if len(vals) == 0 {
			return "1=0", nil
		}
		ph := make([]string, len(vals))
		for i := range vals {
			ph[i] = fmt.Sprintf("$%d", *argIndex)
			*argIndex++
		}
		s := fmt.Sprintf("%s IN (%s)", f, strings.Join(ph, ", "))
		args := make([]any, len(vals))
		copy(args, vals)
		return s, args
	case store.OpBetween:
		r, _ := cond.Value.([2]any)
		s := fmt.Sprintf("%s BETWEEN $%d AND $%d", f, *argIndex, *argIndex+1)
		*argIndex += 2
		return s, []any{r[0], r[1]}
	case store.OpPrefix:
		s := fmt.Sprintf("%s LIKE $%d", f, *argIndex)
		*argIndex++
		return s, []any{fmt.Sprintf("%s%%", cond.Value)}
	case store.OpContains:
		s := fmt.Sprintf("%s LIKE $%d", f, *argIndex)
		*argIndex++
		return s, []any{fmt.Sprintf("%%%s%%", cond.Value)}
	case store.OpIsNull:
		return fmt.Sprintf("%s IS NULL", f), nil
	case store.OpNotNull:
		return fmt.Sprintf("%s IS NOT NULL", f), nil
	default:
		return "", nil
	}
}
