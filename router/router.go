// Package router provides an interface for micro network router
package router

// Router is micro network router
type Router interface {
	// Init initializes the router with options
	Init(...Option) error
	// Options returns the router options
	Options() Options
	// Table returns routing table
	Table() Table
	// Address returns router adddress
	Address() string
	// Gossip returns router gossip address
	Gossip() string
	// Network returns router network address
	Network() string
	// Start starts the router
	Start() error
	// Stop stops the router
	Stop() error
	// String returns debug info
	String() string
}

// Option used by the router
type Option func(*Options)

// RIBOptopn is used to configure RIB
type RIBOption func(*RIBOptions)

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
