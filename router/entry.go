package router

// AddPolicy defines routing table addition policy
type AddPolicy int

const (
	// Override overrides existing routing table route
	OverrideIfExists AddPolicy = iota
	// ErrIfExists returns error if the route already exists
	ErrIfExists
)

// RouteOptions defines micro network routing table route options
type RouteOptions struct {
	// DestAddr is destination address
	DestAddr string
	// Gateway is the next route hop
	Gateway Router
	// Network defines micro network
	Network string
	// Metric is route cost metric
	Metric int
	// Policy defines route addition policy
	Policy AddPolicy
}

// DestAddr sets destination address
func DestAddr(a string) RouteOption {
	return func(o *RouteOptions) {
		o.DestAddr = a
	}
}

// Gateway sets the route gateway
func Gateway(r Router) RouteOption {
	return func(o *RouteOptions) {
		o.Gateway = r
	}
}

// Network sets micro network
func Network(n string) RouteOption {
	return func(o *RouteOptions) {
		o.Network = n
	}
}

// Metric sets route metric
func Metric(m int) RouteOption {
	return func(o *RouteOptions) {
		o.Metric = m
	}
}

// RoutePolicy sets add route policy
func RoutePolicy(p AddPolicy) RouteOption {
	return func(o *RouteOptions) {
		o.Policy = p
	}
}

// Route is routing table route
type Route interface {
	// Options returns route options
	Options() RouteOptions
}

type route struct {
	opts RouteOptions
}

// NewRoute returns new routing table route
func NewRoute(opts ...RouteOption) Route {
	eopts := RouteOptions{}

	for _, o := range opts {
		o(&eopts)
	}

	return &route{
		opts: eopts,
	}
}

// Options returns route options
func (r *route) Options() RouteOptions {
	return r.opts
}
