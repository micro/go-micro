package router

// LookupPolicy defines query policy
type LookupPolicy int

const (
	// DiscardNoRoute discards query when no route is found
	DiscardNoRoute LookupPolicy = iota
	// ClosestMatch returns closest match to supplied query
	ClosestMatch
)

// QueryOptions allow to define routing table query options
type QueryOptions struct {
	// Route allows to set route options
	Route *RouteOptions
	// Service is micro service name
	Service string
	// Policy defines query lookup policy
	Policy LookupPolicy
	// Count defines max number of results to return
	Count int
}

// RouteOpts allows to set the route query options
func RouteOpts(r *RouteOptions) QueryOption {
	return func(o *QueryOptions) {
		o.Route = r
	}
}

// Service allows to set the service name in routing query
func Service(s string) QueryOption {
	return func(o *QueryOptions) {
		o.Service = s
	}
}

// QueryPolicy allows to define query lookup policy
func QueryPolicy(p LookupPolicy) QueryOption {
	return func(o *QueryOptions) {
		o.Policy = p
	}
}

// ResultCount allows to set max results to return
func ResultCount(c int) QueryOption {
	return func(o *QueryOptions) {
		o.Count = c
	}
}

// Query defines routing table query
type Query interface {
	// Options returns query options
	Options() QueryOptions
}

type query struct {
	opts QueryOptions
}

// NewQuery creates new query and returns it
func NewQuery(opts ...QueryOption) Query {
	qopts := QueryOptions{}

	for _, o := range opts {
		o(&qopts)
	}

	return &query{
		opts: qopts,
	}
}

// Options returns query options
func (q *query) Options() QueryOptions {
	return q.opts
}
