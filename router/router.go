// Package router provides a network routing control plane
package router

import (
	"errors"
)

var (
	// DefaultNetwork is default micro network
	DefaultNetwork = "micro"
	// ErrRouteNotFound is returned when no route was found in the routing table
	ErrRouteNotFound = errors.New("route not found")
	// ErrDuplicateRoute is returned when the route already exists
	ErrDuplicateRoute = errors.New("duplicate route")
)

// Router is an interface for a routing control plane
type Router interface {
	// Init initializes the router with options
	Init(...Option) error
	// Options returns the router options
	Options() Options
	// The routing table
	Table() Table
	// Lookup queries routes in the routing table
	Lookup(...QueryOption) ([]Route, error)
	// Watch returns a watcher which tracks updates to the routing table
	Watch(opts ...WatchOption) (Watcher, error)
	// Close the router
	Close() error
	// Returns the router implementation
	String() string
}

// Table is an interface for routing table
type Table interface {
	// Create new route in the routing table
	Create(Route) error
	// Delete existing route from the routing table
	Delete(Route) error
	// Update route in the routing table
	Update(Route) error
	// List all routes in the table
	List() ([]Route, error)
	// Query routes in the routing table
	Query(...QueryOption) ([]Route, error)
}

// Option used by the router
type Option func(*Options)

// StatusCode defines router status
type StatusCode int

const (
	// Running means the router is up and running
	Running StatusCode = iota
	// Stopped means the router has been stopped
	Stopped
	// Error means the router has encountered error
	Error
)
