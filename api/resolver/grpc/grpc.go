// Package grpc resolves a grpc service like /greeter.Say/Hello to greeter service
package grpc

import (
	"errors"
	"net/http"
	"strings"

	"go-micro.dev/v5/api/resolver"
)

// Resolver is the gRPC Resolver.
type Resolver struct{}

// Resolve resolves a http.Request to an grpc Endpoint.
func (r *Resolver) Resolve(req *http.Request) (*resolver.Endpoint, error) {
	// /foo.Bar/Service
	if req.URL.Path == "/" {
		return nil, errors.New("unknown name")
	}
	// [foo.Bar, Service]
	parts := strings.Split(req.URL.Path[1:], "/")
	// [foo, Bar]
	name := strings.Split(parts[0], ".")
	// foo
	return &resolver.Endpoint{
		Name:   strings.Join(name[:len(name)-1], "."),
		Host:   req.Host,
		Method: req.Method,
		Path:   req.URL.Path,
	}, nil
}

// String returns the name of the resolver.
func (r *Resolver) String() string {
	return "grpc"
}

// NewResolver creates a new gRPC resolver.
func NewResolver(opts ...resolver.Option) resolver.Resolver {
	return &Resolver{}
}
