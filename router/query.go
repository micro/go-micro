package router

// QueryOption sets routing table query options
type QueryOption func(*QueryOptions)

// QueryOptions are routing table query options
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
	// Strategy is routing strategy
	Strategy Strategy
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

// QueryStrategy sets strategy to query
func QueryStrategy(s Strategy) QueryOption {
	return func(o *QueryOptions) {
		o.Strategy = s
	}
}

// NewQuery creates new query and returns it
func NewQuery(opts ...QueryOption) QueryOptions {
	// default options
	qopts := QueryOptions{
		Service:  "*",
		Address:  "*",
		Gateway:  "*",
		Network:  "*",
		Router:   "*",
		Strategy: AdvertiseAll,
	}

	for _, o := range opts {
		o(&qopts)
	}

	return qopts
}
