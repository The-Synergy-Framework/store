package sqlstore

import (
	"fmt"
	"strings"

	"store"
)

// CompileMutation compiles a mutation to SQL - simplified implementation
func CompileMutation(tableName string, mutation store.Mutation) (*store.CompiledMutation, error) {
	switch m := mutation.(type) {
	case store.Insert:
		return compileInsert(tableName, m)
	case store.Update:
		return compileUpdate(tableName, m)
	case store.Delete:
		return compileDelete(tableName, m)
	default:
		return nil, fmt.Errorf("unsupported mutation type: %T", mutation)
	}
}

func compileInsert(tableName string, insert store.Insert) (*store.CompiledMutation, error) {
	if len(insert.Values) == 0 {
		return nil, fmt.Errorf("insert values cannot be empty")
	}

	var columns []string
	var placeholders []string
	var args []any

	i := 1
	for col, val := range insert.Values {
		columns = append(columns, col)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		args = append(args, val)
		i++
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	return &store.CompiledMutation{
		SQL:  sql,
		Args: args,
	}, nil
}

func compileUpdate(tableName string, update store.Update) (*store.CompiledMutation, error) {
	if len(update.Set) == 0 {
		return nil, fmt.Errorf("update set values cannot be empty")
	}

	var setParts []string
	var args []any
	i := 1

	// Build SET clause
	for col, val := range update.Set {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", col, i))
		args = append(args, val)
		i++
	}

	sql := fmt.Sprintf("UPDATE %s SET %s", tableName, strings.Join(setParts, ", "))

	// Build WHERE clause if conditions exist
	if len(update.Where) > 0 {
		whereSQL, whereArgs := compileConditions(update.Where, i)
		sql += " WHERE " + whereSQL
		args = append(args, whereArgs...)
	}

	return &store.CompiledMutation{
		SQL:  sql,
		Args: args,
	}, nil
}

func compileDelete(tableName string, delete store.Delete) (*store.CompiledMutation, error) {
	sql := fmt.Sprintf("DELETE FROM %s", tableName)
	var args []any

	// Build WHERE clause if conditions exist
	if len(delete.Where) > 0 {
		whereSQL, whereArgs := compileConditions(delete.Where, 1)
		sql += " WHERE " + whereSQL
		args = append(args, whereArgs...)
	}

	return &store.CompiledMutation{
		SQL:  sql,
		Args: args,
	}, nil
}

// compileConditions compiles a list of conditions to SQL WHERE clause (all ANDed together)
func compileConditions(conditions []store.Condition, startIndex int) (string, []any) {
	if len(conditions) == 0 {
		return "", nil
	}

	var parts []string
	var args []any
	i := startIndex

	for _, cond := range conditions {
		switch cond.Op {
		case store.OpEq:
			parts = append(parts, fmt.Sprintf("%s = $%d", cond.Field, i))
			args = append(args, cond.Value)
			i++
		case store.OpNe:
			parts = append(parts, fmt.Sprintf("%s != $%d", cond.Field, i))
			args = append(args, cond.Value)
			i++
		case store.OpGt:
			parts = append(parts, fmt.Sprintf("%s > $%d", cond.Field, i))
			args = append(args, cond.Value)
			i++
		case store.OpGe:
			parts = append(parts, fmt.Sprintf("%s >= $%d", cond.Field, i))
			args = append(args, cond.Value)
			i++
		case store.OpLt:
			parts = append(parts, fmt.Sprintf("%s < $%d", cond.Field, i))
			args = append(args, cond.Value)
			i++
		case store.OpLe:
			parts = append(parts, fmt.Sprintf("%s <= $%d", cond.Field, i))
			args = append(args, cond.Value)
			i++
		case store.OpIsNull:
			parts = append(parts, fmt.Sprintf("%s IS NULL", cond.Field))
		case store.OpNotNull:
			parts = append(parts, fmt.Sprintf("%s IS NOT NULL", cond.Field))
		case store.OpIn:
			if values, ok := cond.Value.([]any); ok && len(values) > 0 {
				var placeholders []string
				for _, val := range values {
					placeholders = append(placeholders, fmt.Sprintf("$%d", i))
					args = append(args, val)
					i++
				}
				parts = append(parts, fmt.Sprintf("%s IN (%s)", cond.Field, strings.Join(placeholders, ", ")))
			}
		default:
			// For unsupported operators, just do equality
			parts = append(parts, fmt.Sprintf("%s = $%d", cond.Field, i))
			args = append(args, cond.Value)
			i++
		}
	}

	return strings.Join(parts, " AND "), args
}
