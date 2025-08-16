package store

// Mutation is a marker interface for write operations.
type Mutation interface{ isMutation() }

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

// Update represents an update with SET values and a WHERE filter.
type Update struct {
	Set   map[string]any
	Where Node
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

// Delete represents a delete with a WHERE filter.
type Delete struct {
	Where Node
	Hints map[string]any // e.g., {"returning": []string{"id"}}
}

func (Delete) isMutation() {}

func (m Delete) WithReturning(cols ...string) Delete {
	if m.Hints == nil {
		m.Hints = map[string]any{}
	}
	m.Hints["returning"] = cols
	return m
}

// Upsert represents an insert with conflict resolution (dialect-dependent).
// For SQL (PostgreSQL), it compiles to ON CONFLICT (ConflictColumns) DO UPDATE SET UpdateSet.
type Upsert struct {
	Values          map[string]any
	ConflictColumns []string
	UpdateSet       map[string]any // columns to update on conflict
	Hints           map[string]any // e.g., {"returning": []string{"id"}}
}

func (Upsert) isMutation() {}

func (m Upsert) WithReturning(cols ...string) Upsert {
	if m.Hints == nil {
		m.Hints = map[string]any{}
	}
	m.Hints["returning"] = cols
	return m
}

// Helper constructors

func NewInsert(values map[string]any) Insert { return Insert{Values: values} }

func NewUpdate(set map[string]any, where Node) Update { return Update{Set: set, Where: where} }

func NewDelete(where Node) Delete { return Delete{Where: where} }

func NewUpsert(values map[string]any, conflictCols []string, updateSet map[string]any) Upsert {
	return Upsert{Values: values, ConflictColumns: conflictCols, UpdateSet: updateSet}
}
