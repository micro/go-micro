// Package router provides an interface for micro network routers
package router

import (
	"errors"

	"github.com/micro/go-micro/registry"
)

// Router is micro network router
type Router interface {
	// Initi initializes Router with options
	Init(...Option) error
	// Options returns Router options
	Options() Options
	// AddRoute adds new service route
	AddRoute(*Route, ...RouteOption) error
	// RemoveRoute removes service route
	RemoveRoute(*Route) error
	// GetRoute returns list of routes for service
	GetRoute(*Service) ([]*Route, error)
	// List returns all routes
	List() ([]*Route, error)
	// String implemens fmt.Stringer interface
	String() string
}

// Option used by the Router
type Option func(*Options)

var (
	DefaultRouter = NewRouter()

	// Not found error when Get is called
	ErrNotFound = errors.New("route not found")
)

// NewRouter creates new Router and returns it
func NewRouter(opts ...Option) Router {
	// set Registry to DefaultRegistry
	opt := Options{
		Registry: registry.DefaultRegistry,
	}

	for _, o := range opts {
		o(&opt)
	}

	return newRouter(opts...)
}
