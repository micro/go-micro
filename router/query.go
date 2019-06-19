package router

// LookupPolicy defines query policy
type LookupPolicy int

const (
	// DiscardNoRoute discards query when no route is found
	DiscardNoRoute LookupPolicy = iota
	// ClosestMatch returns closest match to supplied query
	ClosestMatch
)

// String returns human representation of LookupPolicy
func (lp LookupPolicy) String() string {
	switch lp {
	case DiscardNoRoute:
		return "DISCARD"
	case ClosestMatch:
		return "CLOSEST"
	default:
		return "UNKNOWN"
	}
}

// QueryOption sets routing table query options
type QueryOption func(*QueryOptions)

// QueryOptions are routing table query options
type QueryOptions struct {
	// Destination is destination address
	Destination string
	// Network is network address
	Network string
	// Router is gateway address
	Router Router
	// Metric is route metric
	Metric int
	// Policy is query lookup policy
	Policy LookupPolicy
}

// QueryDestination sets query destination address
func QueryDestination(a string) QueryOption {
	return func(o *QueryOptions) {
		o.Destination = a
	}
}

// QueryNetwork sets query network address
func QueryNetwork(a string) QueryOption {
	return func(o *QueryOptions) {
		o.Network = a
	}
}

// QueryRouter sets query gateway address
func QueryRouter(r Router) QueryOption {
	return func(o *QueryOptions) {
		o.Router = r
	}
}

// QueryMetric sets query metric
func QueryMetric(m int) QueryOption {
	return func(o *QueryOptions) {
		o.Metric = m
	}
}

// QueryPolicy sets query policy
// NOTE: this might be renamed to filter or some such
func QueryPolicy(p LookupPolicy) QueryOption {
	return func(o *QueryOptions) {
		o.Policy = p
	}
}

// Query is routing table query
type Query interface {
	// Options returns query options
	Options() QueryOptions
}

// query is a basic implementation of Query
type query struct {
	opts QueryOptions
}

// NewQuery creates new query and returns it
func NewQuery(opts ...QueryOption) Query {
	// default gateway for wildcard router
	r := newRouter(ID("*"))

	// default options
	// NOTE: by default we use DefaultNetworkMetric
	qopts := QueryOptions{
		Destination: "*",
		Network:     "*",
		Router:      r,
		Metric:      DefaultNetworkMetric,
		Policy:      DiscardNoRoute,
	}

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
