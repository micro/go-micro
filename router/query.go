package router

// QueryOption sets routing table query options
type QueryOption func(*QueryOptions)

// QueryOptions are routing table query options
// TODO replace with Filter(Route) bool
type QueryOptions struct {
	// Service is destination service name
	Service string
	// Address of the service
	Address string
	// Gateway is route gateway
	Gateway string
	// Network is network address
	Network string
	// Router is router id
	Router string
	// Link to query
	Link string
}

// QueryService sets service to query
func QueryService(s string) QueryOption {
	return func(o *QueryOptions) {
		o.Service = s
	}
}

// QueryAddress sets service to query
func QueryAddress(a string) QueryOption {
	return func(o *QueryOptions) {
		o.Address = a
	}
}

// QueryGateway sets gateway address to query
func QueryGateway(g string) QueryOption {
	return func(o *QueryOptions) {
		o.Gateway = g
	}
}

// QueryNetwork sets network name to query
func QueryNetwork(n string) QueryOption {
	return func(o *QueryOptions) {
		o.Network = n
	}
}

// QueryRouter sets router id to query
func QueryRouter(r string) QueryOption {
	return func(o *QueryOptions) {
		o.Router = r
	}
}

// QueryLink sets the link to query
func QueryLink(link string) QueryOption {
	return func(o *QueryOptions) {
		o.Link = link
	}
}

// NewQuery creates new query and returns it
func NewQuery(opts ...QueryOption) QueryOptions {
	// default options
	qopts := QueryOptions{
		Service: "*",
		Address: "*",
		Gateway: "*",
		Network: "*",
		Router:  "*",
		Link:    DefaultLink,
	}

	for _, o := range opts {
		o(&qopts)
	}

	return qopts
}
