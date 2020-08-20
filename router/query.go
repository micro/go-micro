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

// isMatch checks if the route matches given query options
func isMatch(route Route, address, gateway, network, rtr, link string) bool {
	// matches the values provided
	match := func(a, b string) bool {
		if a == "*" || b == "*" || a == b {
			return true
		}
		return false
	}

	// a simple struct to hold our values
	type compare struct {
		a string
		b string
	}

	// compare the following values
	values := []compare{
		{gateway, route.Gateway},
		{network, route.Network},
		{rtr, route.Router},
		{address, route.Address},
		{link, route.Link},
	}

	for _, v := range values {
		// attempt to match each value
		if !match(v.a, v.b) {
			return false
		}
	}

	return true
}

// filterRoutes finds all the routes for given network and router and returns them
func Filter(routes []Route, opts QueryOptions) []Route {
	address := opts.Address
	gateway := opts.Gateway
	network := opts.Network
	rtr := opts.Router
	link := opts.Link

	// routeMap stores the routes we're going to advertise
	routeMap := make(map[string][]Route)

	for _, route := range routes {
		if isMatch(route, address, gateway, network, rtr, link) {
			// add matchihg route to the routeMap
			routeKey := route.Service + "@" + route.Network
			routeMap[routeKey] = append(routeMap[routeKey], route)
		}
	}

	var results []Route

	for _, route := range routeMap {
		results = append(results, route...)
	}

	return results
}
