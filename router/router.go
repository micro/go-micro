// Package router provides an interface for micro network router
package router

// Router is micro network router
type Router interface {
	// Init initializes the router with options
	Init(...Option) error
	// Options returns the router options
	Options() Options
	// ID returns the id of the router
	ID() string
	// Table returns the routing table
	Table() Table
	// Address returns the router adddress
	Address() string
	// Network returns the network address of the router
	Network() string
	// Advertise starts advertising the routes to the network
	Advertise() error
	// Stop stops the router
	Stop() error
	// String returns debug info
	String() string
}

// Option used by the router
type Option func(*Options)

// NewRouter creates new Router and returns it
func NewRouter(opts ...Option) Router {
	return newRouter(opts...)
}
