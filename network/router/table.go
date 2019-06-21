package router

import (
	"errors"
)

var (
	// ErrRouteNotFound is returned when no route was found in the routing table
	ErrRouteNotFound = errors.New("route not found")
	// ErrDuplicateRoute is returned when the route already exists
	ErrDuplicateRoute = errors.New("duplicate route")
)

// Table defines routing table interface
type Table interface {
	// Init initializes the router with options
	Init(...TableOption) error
	// Options returns the router options
	Options() TableOptions
	// Add adds new route to the routing table
	Add(Route) error
	// Delete deletes existing route from the routing table
	Delete(Route) error
	// Update updates route in the routing table
	Update(Route) error
	// List returns the list of all routes in the table
	List() ([]Route, error)
	// Lookup looks up routes in the routing table and returns them
	Lookup(Query) ([]Route, error)
	// Watch returns a watcher which allows to track updates to the routing table
	Watch(opts ...WatchOption) (Watcher, error)
	// Size returns the size of the routing table
	Size() int
	// String prints the routing table
	String() string
}

// TableOption used by the routing table
type TableOption func(*TableOptions)

// NewTable creates new routing table and returns it
func NewTable(opts ...TableOption) Table {
	return newTable(opts...)
}
