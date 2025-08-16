package store

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
	OpNotIn    Operator = "not_in"
	OpBetween  Operator = "between"
	OpPrefix   Operator = "prefix"   // string starts with
	OpSuffix   Operator = "suffix"   // string ends with
	OpContains Operator = "contains" // string contains
	OpLike     Operator = "like"     // SQL LIKE pattern
	OpILike    Operator = "ilike"    // case-insensitive LIKE
	OpRegex    Operator = "regex"    // regular expression match
	OpIsNull   Operator = "isnull"
	OpNotNull  Operator = "notnull"
)

// Condition is a simple filter condition (field op value).
type Condition struct {
	Field string
	Op    Operator
	// Value can be a single value, []any for OpIn, or [2]any for OpBetween.
	Value any
}

// Order defines ordering on a field.
type Order struct {
	Field string
	Desc  bool
}

// Helper functions for creating conditions
func Eq(field string, value any) Condition {
	return Condition{Field: field, Op: OpEq, Value: value}
}

func Ne(field string, value any) Condition {
	return Condition{Field: field, Op: OpNe, Value: value}
}

func Gt(field string, value any) Condition {
	return Condition{Field: field, Op: OpGt, Value: value}
}

func Ge(field string, value any) Condition {
	return Condition{Field: field, Op: OpGe, Value: value}
}

func Lt(field string, value any) Condition {
	return Condition{Field: field, Op: OpLt, Value: value}
}

func Le(field string, value any) Condition {
	return Condition{Field: field, Op: OpLe, Value: value}
}

func In(field string, values ...any) Condition {
	return Condition{Field: field, Op: OpIn, Value: values}
}

func NotIn(field string, values ...any) Condition {
	return Condition{Field: field, Op: OpNotIn, Value: values}
}

func Between(field string, from, to any) Condition {
	return Condition{Field: field, Op: OpBetween, Value: [2]any{from, to}}
}

func Contains(field string, value string) Condition {
	return Condition{Field: field, Op: OpContains, Value: value}
}

func Like(field string, pattern string) Condition {
	return Condition{Field: field, Op: OpLike, Value: pattern}
}

func IsNull(field string) Condition {
	return Condition{Field: field, Op: OpIsNull, Value: nil}
}

func NotNull(field string) Condition {
	return Condition{Field: field, Op: OpNotNull, Value: nil}
}

// Helper functions for creating orders
func Asc(field string) Order {
	return Order{Field: field, Desc: false}
}

func Desc(field string) Order {
	return Order{Field: field, Desc: true}
}
