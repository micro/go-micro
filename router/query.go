package router

// LookupPolicy defines query policy
type LookupPolicy int

const (
	// DiscardNoRoute discards query when no route is found
	DiscardNoRoute LookupPolicy = iota
	// ClosestMatch returns closest match to supplied query
	ClosestMatch
)

// QueryOption is used to define query options
type QueryOption func(*QueryOptions)

// QueryOptions allow to define routing table query options
type QueryOptions struct {
	// DestAddr defines destination address
	DestAddr string
	// NetworkAddress defines network address
	Network string
	// Policy defines query lookup policy
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

// QueryPolicy allows to define query lookup policy
func QueryPolicy(p LookupPolicy) QueryOption {
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
	// default options
	qopts := QueryOptions{
		DestAddr: "*",
		Network:  "*",
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
