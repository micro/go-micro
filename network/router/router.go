// Package router provides a network routing control plane
package router

// Router is an interface for a routing control plane
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

var (
	DefaultRouter = NewRouter()
)

// NewRouter creates new Router and returns it
func NewRouter(opts ...Option) Router {
	return newRouter(opts...)
}
