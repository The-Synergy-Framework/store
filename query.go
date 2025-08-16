package store

import "context"

// Operator represents a comparison operation in filters.
type Operator string

const (
	OpEq       Operator = "eq"
	OpNe       Operator = "ne"
	OpGt       Operator = "gt"
	OpGe       Operator = "ge"
	OpLt       Operator = "lt"
	OpLe       Operator = "le"
	OpIn       Operator = "in"
	OpBetween  Operator = "between"
	OpPrefix   Operator = "prefix"   // string starts with
	OpContains Operator = "contains" // string contains
	OpIsNull   Operator = "isnull"
	OpNotNull  Operator = "notnull"
)

// Node is a filter expression node.
type Node interface{ isNode() }

// Condition is a leaf filter (field op value).
type Condition struct {
	Field string
	Op    Operator
	// Value can be a single value, []any for OpIn, or [2]any for OpBetween.
	Value any
}

func (Condition) isNode() {}

// And groups child expressions with logical AND.
type And struct{ Children []Node }

func (And) isNode() {}

// Or groups child expressions with logical OR.
type Or struct{ Children []Node }

func (Or) isNode() {}

// Order defines ordering on a field.
type Order struct {
	Field string
	Desc  bool
}

// Query captures a backend-agnostic query intent.
type Query struct {
	SelectFields []string
	Filter       Node
	OrderBy      []Order
	Limit        *int
	Offset       *int
	PageSize     *int32
	Cursor       string
	// Hints carry backend-specific options (optional).
	Hints map[string]any
}

// Builder provides a fluent API to construct a Query.
type Builder struct{ q Query }

func New() *Builder { return &Builder{q: Query{}} }

func (b *Builder) Select(columns ...string) *Builder {
	b.q.SelectFields = append([]string{}, columns...)
	return b
}

func (b *Builder) Where(node Node) *Builder { b.q.Filter = node; return b }

func (b *Builder) And(nodes ...Node) *Builder { b.q.Filter = And{Children: nodes}; return b }

func (b *Builder) Or(nodes ...Node) *Builder { b.q.Filter = Or{Children: nodes}; return b }

func (b *Builder) OrderBy(field string, desc bool) *Builder {
	b.q.OrderBy = append(b.q.OrderBy, Order{Field: field, Desc: desc})
	return b
}

func (b *Builder) OrderByAsc(field string) *Builder  { return b.OrderBy(field, false) }
func (b *Builder) OrderByDesc(field string) *Builder { return b.OrderBy(field, true) }

func (b *Builder) Limit(n int) *Builder  { b.q.Limit = &n; return b }
func (b *Builder) Offset(n int) *Builder { b.q.Offset = &n; return b }

func (b *Builder) Page(size int32, cursor string) *Builder {
	b.q.PageSize = &size
	b.q.Cursor = cursor
	return b
}

func (b *Builder) Hint(key string, value any) *Builder {
	if b.q.Hints == nil {
		b.q.Hints = map[string]any{}
	}
	b.q.Hints[key] = value
	return b
}

func (b *Builder) Build() Query { return b.q }

// Helper constructors for conditions.
func Eq(field string, value any) Node    { return Condition{Field: field, Op: OpEq, Value: value} }
func Ne(field string, value any) Node    { return Condition{Field: field, Op: OpNe, Value: value} }
func Gt(field string, value any) Node    { return Condition{Field: field, Op: OpGt, Value: value} }
func Ge(field string, value any) Node    { return Condition{Field: field, Op: OpGe, Value: value} }
func Lt(field string, value any) Node    { return Condition{Field: field, Op: OpLt, Value: value} }
func Le(field string, value any) Node    { return Condition{Field: field, Op: OpLe, Value: value} }
func In(field string, values []any) Node { return Condition{Field: field, Op: OpIn, Value: values} }

func Between(field string, from, to any) Node {
	return Condition{Field: field, Op: OpBetween, Value: [2]any{from, to}}
}

func Prefix(field string, prefix string) Node {
	return Condition{Field: field, Op: OpPrefix, Value: prefix}
}

func Contains(field string, substr string) Node {
	return Condition{Field: field, Op: OpContains, Value: substr}
}

func IsNull(field string) Node  { return Condition{Field: field, Op: OpIsNull} }
func NotNull(field string) Node { return Condition{Field: field, Op: OpNotNull} }

// Context key helpers (optional) for backends that want query-scoped settings.
type ctxKey struct{}

func WithHint(ctx context.Context, key string, value any) context.Context {
	m, _ := ctx.Value(ctxKey{}).(map[string]any)
	if m == nil {
		m = map[string]any{}
	}
	m[key] = value
	return context.WithValue(ctx, ctxKey{}, m)
}

func HintsFromContext(ctx context.Context) map[string]any {
	m, _ := ctx.Value(ctxKey{}).(map[string]any)
	return m
}
