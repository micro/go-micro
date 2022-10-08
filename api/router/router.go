// Package router provides api service routing
package router

import (
	"net/http"

	"go-micro.dev/v4/registry"
)

// Router is used to determine an endpoint for a request.
type Router interface {
	// Returns options
	Options() Options
	// Register endpoint in router
	Register(r *Route) error
	// Deregister endpoint from router
	Deregister(r *Route) error
	// Route returns an api.Service route
	Route(r *http.Request) (*Route, error)
	// Stop the router
	Stop() error
}

type Route struct {
	// Name of service
	Service string
	// The endpoint for this service
	Endpoint *Endpoint
	// Versions of this service
	Versions []*registry.Service
}

// Endpoint is a mapping between an RPC method and HTTP endpoint.
type Endpoint struct {
	// RPC Method e.g. Greeter.Hello
	Name string
	// What the endpoint is for
	Description string
	// API Handler e.g rpc, proxy
	Handler string
	// HTTP Host e.g example.com
	Host []string
	// HTTP Methods e.g GET, POST
	Method []string
	// HTTP Path e.g /greeter. Expect POSIX regex
	Path []string
	// Stream flag
	Stream bool
}
