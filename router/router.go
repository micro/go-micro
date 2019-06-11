// Package router provides an interface for micro network router
package router

// Router is micro network router
type Router interface {
	// Initi initializes Router with options
	Init(...Option) error
	// Options returns Router options
	Options() Options
	// Table returns routing table
	Table() Table
	// Address returns router adddress
	Address() string
	// Network returns router network address
	Network() string
	// Start starts router
	Start() error
	// Stop stops router
	Stop() error
	// String returns router debug info
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

// WatchOption is used to define what routes to watch in the table
type WatchOption func(*WatchOptions)

// NewRouter creates new Router and returns it
func NewRouter(opts ...Option) Router {
	return newRouter(opts...)
}
