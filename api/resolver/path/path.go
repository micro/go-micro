// Package path resolves using http path
package path

import (
	"net/http"
	"strings"

	"go-micro.dev/v5/api/resolver"
)

// Resolver is a path resolver.
type Resolver struct {
	opts resolver.Options
}

// Resolve resolves a http.Request to an grpc Endpoint.
func (r *Resolver) Resolve(req *http.Request) (*resolver.Endpoint, error) {
	if req.URL.Path == "/" {
		return nil, resolver.ErrNotFound
	}

	parts := strings.Split(req.URL.Path[1:], "/")
	ns := r.opts.Namespace(req)

	return &resolver.Endpoint{
		Name:   ns + "." + parts[0],
		Host:   req.Host,
		Method: req.Method,
		Path:   req.URL.Path,
	}, nil
}

func (r *Resolver) String() string {
	return "path"
}

// NewResolver returns a new path resolver.
func NewResolver(opts ...resolver.Option) resolver.Resolver {
	return &Resolver{opts: resolver.NewOptions(opts...)}
}
