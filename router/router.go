// Package router provides an interface for micro network routers
package router

var (
	// DefaultRouter returns default micro router
	DefaultRouter = NewRouter()
)

// Router is micro network router
type Router interface {
	// Initi initializes Router with options
	Init(...Option) error
	// Options returns Router options
	Options() Options
	// Add adds new entry into routing table
	Add(*Entry, ...RouteOption) error
	// Remove removes entry from the routing table
	Remove(*Entry) error
	// Update updates entry in the routing table
	Update(*Entry) error
	// Lookup queries the routing table and returns matching entries
	Lookup(Query) ([]*Entry, error)
	// Table returns routing table
	Table() Table
	// Address is Router adddress
	Address() string
	// Network defines network router is in
	Network() string
	// String implemens fmt.Stringer interface
	String() string
}

// RIB is Routing Information Base
type RIB interface {
	// String returns debug info
	String() string
}

// Option used by the Router
type Option func(*Options)

// RouteOption is used by Router for adding routing table entries
type RouteOption func(*RouteOptions)

// QueryOption is used to defined routing table lookup query
type QueryOption func(*QueryOptions)

// NewRouter creates new Router and returns it
func NewRouter(opts ...Option) Router {
	// set default options
	ropts := Options{
		Table: DefaultTable,
	}

	for _, o := range opts {
		o(&ropts)
	}

	return newRouter(opts...)
}
