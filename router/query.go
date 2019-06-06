package router

// Policy defines query policy
type QueryPolicy int

const (
	// DiscardNoRoute discards query when no rout is found
	DiscardNoRoute QueryPolicy = iota
	// ClosestMatch returns closest match to query
	ClosestMatch
)

// QueryOptions allow to define routing table query options
type QueryOptions struct {
	// Route allows to set route options
	Route *RouteOptions
	// Service is micro service name
	Service string
	// Policy defines query lookup policy
	Policy QueryPolicy
}

// Route allows to set the route query options
func Route(r *RouteOptions) QueryOption {
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

// Policy allows to define query lookup policy
func Policy(p QueryPolicy) QueryOption {
	return func(o *QueryOptions) {
		o.Policy = p
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
