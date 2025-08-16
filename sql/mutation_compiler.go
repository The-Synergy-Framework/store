package sqlstore

import (
	"fmt"
	"sort"
	"strings"

	"store"
)

// CompileMutation compiles a mutation for a given table to SQL and args.
func CompileMutation(table string, m store.Mutation) (*CompiledSQL, error) {
	switch mt := m.(type) {
	case store.Insert:
		return compileInsert(table, mt)
	case store.Update:
		return compileUpdate(table, mt)
	case store.Delete:
		return compileDelete(table, mt)
	case store.Upsert:
		return compileUpsert(table, mt)
	default:
		return nil, fmt.Errorf("unsupported mutation type")
	}
}

func compileInsert(table string, m store.Insert) (*CompiledSQL, error) {
	if len(m.Values) == 0 {
		return nil, fmt.Errorf("insert has no values")
	}
	cols := make([]string, 0, len(m.Values))
	for k := range m.Values {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	ph := make([]string, len(cols))
	args := make([]any, len(cols))
	for i, c := range cols {
		ph[i] = fmt.Sprintf("$%d", i+1)
		args[i] = m.Values[c]
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ", "), strings.Join(ph, ", "))
	if r, ok := returningFromHints(m.Hints); ok {
		sql += " RETURNING " + strings.Join(r, ", ")
	}
	return &CompiledSQL{SQL: sql, Args: args}, nil
}

func compileUpdate(table string, m store.Update) (*CompiledSQL, error) {
	if len(m.Set) == 0 {
		return nil, fmt.Errorf("update has no set values")
	}
	// SET clause
	setCols := make([]string, 0, len(m.Set))
	for k := range m.Set {
		setCols = append(setCols, k)
	}
	sort.Strings(setCols)
	setParts := make([]string, len(setCols))
	args := make([]any, len(setCols))
	idx := 1
	for i, c := range setCols {
		setParts[i] = fmt.Sprintf("%s = $%d", c, idx)
		args[i] = m.Set[c]
		idx++
	}
	sql := fmt.Sprintf("UPDATE %s SET %s", table, strings.Join(setParts, ", "))
	// WHERE
	if m.Where != nil {
		wsql, wargs := compileWhere(m.Where, &idx)
		if wsql != "" {
			sql += " WHERE " + wsql
			args = append(args, wargs...)
		}
	}
	if r, ok := returningFromHints(m.Hints); ok {
		sql += " RETURNING " + strings.Join(r, ", ")
	}
	return &CompiledSQL{SQL: sql, Args: args}, nil
}

func compileDelete(table string, m store.Delete) (*CompiledSQL, error) {
	sql := fmt.Sprintf("DELETE FROM %s", table)
	args := []any{}
	idx := 1
	if m.Where != nil {
		wsql, wargs := compileWhere(m.Where, &idx)
		if wsql != "" {
			sql += " WHERE " + wsql
			args = append(args, wargs...)
		}
	}
	if r, ok := returningFromHints(m.Hints); ok {
		sql += " RETURNING " + strings.Join(r, ", ")
	}
	return &CompiledSQL{SQL: sql, Args: args}, nil
}

func compileUpsert(table string, m store.Upsert) (*CompiledSQL, error) {
	if len(m.Values) == 0 {
		return nil, fmt.Errorf("upsert has no values")
	}
	cols := make([]string, 0, len(m.Values))
	for k := range m.Values {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	ph := make([]string, len(cols))
	args := make([]any, len(cols))
	for i, c := range cols {
		ph[i] = fmt.Sprintf("$%d", i+1)
		args[i] = m.Values[c]
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ", "), strings.Join(ph, ", "))
	if len(m.ConflictColumns) > 0 {
		sql += fmt.Sprintf(" ON CONFLICT (%s)", strings.Join(m.ConflictColumns, ", "))
		if len(m.UpdateSet) > 0 {
			setCols := make([]string, 0, len(m.UpdateSet))
			for k := range m.UpdateSet {
				setCols = append(setCols, k)
			}
			sort.Strings(setCols)
			parts := make([]string, len(setCols))
			idx := len(args) + 1
			for i, c := range setCols {
				parts[i] = fmt.Sprintf("%s = $%d", c, idx)
				args = append(args, m.UpdateSet[c])
				idx++
			}
			sql += " DO UPDATE SET " + strings.Join(parts, ", ")
		} else {
			sql += " DO NOTHING"
		}
	}
	if r, ok := returningFromHints(m.Hints); ok {
		sql += " RETURNING " + strings.Join(r, ", ")
	}
	return &CompiledSQL{SQL: sql, Args: args}, nil
}

func compileWhere(n store.Node, idx *int) (string, []any) {
	comp := NewSQLCompiler("")
	return comp.compileNode(n, idx)
}

func returningFromHints(h map[string]any) ([]string, bool) {
	if len(h) == 0 {
		return nil, false
	}
	if v, ok := h["returning"]; ok {
		if cols, ok2 := v.([]string); ok2 && len(cols) > 0 {
			return cols, true
		}
	}
	return nil, false
}
