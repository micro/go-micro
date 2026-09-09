package model

// QueryOptions configures a List or Count operation.
type QueryOptions struct {
	Filters []Filter
	OrderBy string
	Desc    bool
	Limit   uint
	Offset  uint
}

// Filter represents a field-level query condition.
type Filter struct {
	Field string // Column name
	Op    string // Operator: =, !=, <, >, <=, >=, LIKE
	Value any    // Comparison value
}

// QueryOption sets values in QueryOptions.
type QueryOption func(*QueryOptions)

// ApplyQueryOptions applies a set of QueryOptions and returns the result.
func ApplyQueryOptions(opts ...QueryOption) QueryOptions {
	q := QueryOptions{}
	for _, o := range opts {
		o(&q)
	}
	return q
}

// Where adds an equality filter: field = value.
func Where(field string, value any) QueryOption {
	return func(q *QueryOptions) {
		q.Filters = append(q.Filters, Filter{Field: field, Op: "=", Value: value})
	}
}

// WhereOp adds a filter with a custom operator (=, !=, <, >, <=, >=, LIKE).
func WhereOp(field, op string, value any) QueryOption {
	return func(q *QueryOptions) {
		q.Filters = append(q.Filters, Filter{Field: field, Op: op, Value: value})
	}
}

// OrderAsc orders results by field ascending.
func OrderAsc(field string) QueryOption {
	return func(q *QueryOptions) {
		q.OrderBy = field
		q.Desc = false
	}
}

// OrderDesc orders results by field descending.
func OrderDesc(field string) QueryOption {
	return func(q *QueryOptions) {
		q.OrderBy = field
		q.Desc = true
	}
}

// Limit limits the number of returned records.
func Limit(n uint) QueryOption {
	return func(q *QueryOptions) {
		q.Limit = n
	}
}

// Offset skips the first n records (for pagination).
func Offset(n uint) QueryOption {
	return func(q *QueryOptions) {
		q.Offset = n
	}
}
