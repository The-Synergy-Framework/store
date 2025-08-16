package store

// Mutation is a marker interface for write operations.
type Mutation interface{ isMutation() }

// CompiledMutation represents a backend-specific compiled mutation.
type CompiledMutation struct {
	SQL  string
	Args []any
	// Hints for mutation-specific options (e.g., RETURNING clauses)
	Hints map[string]any
}

// Insert represents an insert operation with column values.
type Insert struct {
	Values map[string]any
	Hints  map[string]any // e.g., {"returning": []string{"id"}}
}

func (Insert) isMutation() {}

func (m Insert) WithReturning(cols ...string) Insert {
	if m.Hints == nil {
		m.Hints = map[string]any{}
	}
	m.Hints["returning"] = cols
	return m
}

// Update represents an update with SET values and WHERE conditions.
type Update struct {
	Set   map[string]any
	Where []Condition    // Simple list of conditions (all ANDed together)
	Hints map[string]any // e.g., {"returning": []string{"updated_at"}}
}

func (Update) isMutation() {}

func (m Update) WithReturning(cols ...string) Update {
	if m.Hints == nil {
		m.Hints = map[string]any{}
	}
	m.Hints["returning"] = cols
	return m
}

// Delete represents a delete with WHERE conditions.
type Delete struct {
	Where []Condition // Simple list of conditions (all ANDed together)
	Hints map[string]any
}

func (Delete) isMutation() {}

func (m Delete) WithReturning(cols ...string) Delete {
	if m.Hints == nil {
		m.Hints = map[string]any{}
	}
	m.Hints["returning"] = cols
	return m
}

// MutationResult represents the result of a mutation operation.
type MutationResult struct {
	RowsAffected int64
	LastInsertID string
	Returning    []map[string]any
}

// Helper constructors for mutations
func NewInsert(values map[string]any) Insert {
	return Insert{Values: values}
}

func NewUpdate(set map[string]any, conditions ...Condition) Update {
	return Update{Set: set, Where: conditions}
}

func NewDelete(conditions ...Condition) Delete {
	return Delete{Where: conditions}
}
