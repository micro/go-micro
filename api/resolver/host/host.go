// Package host resolves using http host
package host

import (
	"net/http"

	"go-micro.dev/v5/api/resolver"
)

// Resolver is a host resolver.
type Resolver struct {
	opts resolver.Options
}

// Resolve resolves a http.Request to an grpc Endpoint.
func (r *Resolver) Resolve(req *http.Request) (*resolver.Endpoint, error) {
	return &resolver.Endpoint{
		Name:   req.Host,
		Host:   req.Host,
		Method: req.Method,
		Path:   req.URL.Path,
	}, nil
}

// String returns the name of the resolver.
func (r *Resolver) String() string {
	return "host"
}

// NewResolver creates a new host resolver.
func NewResolver(opts ...resolver.Option) resolver.Resolver {
	return &Resolver{opts: resolver.NewOptions(opts...)}
}
