package router

// LookupOption sets routing table query options
type LookupOption func(*LookupOptions)

// LookupOptions are routing table query options
// TODO replace with Filter(Route) bool
type LookupOptions struct {
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

// LookupAddress sets service to query
func LookupAddress(a string) LookupOption {
	return func(o *LookupOptions) {
		o.Address = a
	}
}

// LookupGateway sets gateway address to query
func LookupGateway(g string) LookupOption {
	return func(o *LookupOptions) {
		o.Gateway = g
	}
}

// LookupNetwork sets network name to query
func LookupNetwork(n string) LookupOption {
	return func(o *LookupOptions) {
		o.Network = n
	}
}

// LookupRouter sets router id to query
func LookupRouter(r string) LookupOption {
	return func(o *LookupOptions) {
		o.Router = r
	}
}

// LookupLink sets the link to query
func LookupLink(link string) LookupOption {
	return func(o *LookupOptions) {
		o.Link = link
	}
}

// NewLookup creates new query and returns it
func NewLookup(opts ...LookupOption) LookupOptions {
	// default options
	qopts := LookupOptions{
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
func Filter(routes []Route, opts LookupOptions) []Route {
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
