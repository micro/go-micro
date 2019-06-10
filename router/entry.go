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
	// Hop is the next route hop
	Hop Router
	// SrcAddr defines local routing address
	// On local networkss, this will be the address of local router
	SrcAddr string
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

// Hop allows to set the route route options
func Hop(r Router) RouteOption {
	return func(o *RouteOptions) {
		o.Hop = r
	}
}

// SrcAddr sets source address
func SrcAddr(a string) RouteOption {
	return func(o *RouteOptions) {
		o.SrcAddr = a
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
