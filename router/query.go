package router

// LookupPolicy defines query policy
type LookupPolicy int

const (
	// DiscardNoRoute discards query when no route is found
	DiscardNoRoute LookupPolicy = iota
	// ClosestMatch returns closest match to supplied query
	ClosestMatch
)

// QueryOption sets routing table query options
type QueryOption func(*QueryOptions)

// QueryOptions are routing table query options
type QueryOptions struct {
	// DestAddr is destination address
	DestAddr string
	// NetworkAddress is network address
	Network string
	// Gateway is gateway address
	Gateway Router
	// Policy is query lookup policy
	Policy LookupPolicy
}

// QueryDestAddr sets query destination address
func QueryDestAddr(a string) QueryOption {
	return func(o *QueryOptions) {
		o.DestAddr = a
	}
}

// QueryNetwork sets query network address
func QueryNetwork(a string) QueryOption {
	return func(o *QueryOptions) {
		o.Network = a
	}
}

// QueryGateway sets query gateway address
func QueryGateway(r Router) QueryOption {
	return func(o *QueryOptions) {
		o.Gateway = r
	}
}

// QueryPolicy sets query policy
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
	qopts := QueryOptions{
		DestAddr: "*",
		Network:  "*",
		Gateway:  r,
		Policy:   DiscardNoRoute,
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
