// Package router provides an interface for micro network routers
package router

// Router is micro network router
type Router interface {
	// Initi initializes Router with options
	Init(...Option) error
	// Options returns Router options
	Options() Options
	// Table returns routing table
	Table() Table
	// Address returns router gossip adddress
	Address() string
	// Network returns micro network address
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
	return newRouter(opts...)
}
