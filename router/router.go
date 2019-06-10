// Package router provides an interface for micro network routers
package router

// Router is micro network router
type Router interface {
	// Initi initializes Router with options
	Init(...Option) error
	// Options returns Router options
	Options() Options
	// Add adds new entry into routing table
	Add(Route) error
	// Remove removes entry from the routing table
	Remove(Route) error
	// Update updates entry in the routing table
	Update(...RouteOption) error
	// Lookup queries the routing table and returns matching entries
	Lookup(Query) ([]*Route, error)
	// Table returns routing table
	Table() Table
	// Address returns the router bind adddress
	Address() string
	// Network returns router's micro network bind address
	Network() string
	// String implemens fmt.Stringer interface
	String() string
}

// RIB is Routing Information Base
type RIB interface {
	// String returns debug info
	String() string
}

// Option used by the router
type Option func(*Options)

// RouteOption is used to define routing table entry options
type RouteOption func(*RouteOptions)

// QueryOption is used to define query options
type QueryOption func(*QueryOptions)

// NewRouter creates new Router and returns it
func NewRouter(opts ...Option) Router {
	// set default options
	ropts := Options{
		// Default table
		Table: NewTable(),
	}

	for _, o := range opts {
		o(&ropts)
	}

	return newRouter(opts...)
}
