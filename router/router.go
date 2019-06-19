// Package router provides an interface for micro network router
package router

// Router is micro network router
type Router interface {
	// Init initializes the router with options
	Init(...Option) error
	// Options returns the router options
	Options() Options
	// ID returns id of the router
	ID() string
	// Table returns the router routing table
	Table() Table
	// Address returns the router adddress
	Address() string
	// Network returns the router network address
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

// NewRouter creates new Router and returns it
func NewRouter(opts ...Option) Router {
	return newRouter(opts...)
}
