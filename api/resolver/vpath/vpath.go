// Package vpath resolves using http path and recognized versioned urls
package vpath

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"go-micro.dev/v4/api/resolver"
)

// NewResolver returns a new vpath resolver.
func NewResolver(opts ...resolver.Option) resolver.Resolver {
	return &Resolver{opts: resolver.NewOptions(opts...)}
}

// Resolver is a vpath resolver.
type Resolver struct {
	opts resolver.Options
}

var (
	re = regexp.MustCompile("^v[0-9]+$")
)

// Resolve resolves a http.Request to an grpc Endpoint.
func (r *Resolver) Resolve(req *http.Request) (*resolver.Endpoint, error) {
	if req.URL.Path == "/" {
		return nil, errors.New("unknown name")
	}

	parts := strings.Split(req.URL.Path[1:], "/")
	if len(parts) == 1 {
		return &resolver.Endpoint{
			Name:   r.withNamespace(req, parts...),
			Host:   req.Host,
			Method: req.Method,
			Path:   req.URL.Path,
		}, nil
	}

	// /v1/foo
	if re.MatchString(parts[0]) {
		return &resolver.Endpoint{
			Name:   r.withNamespace(req, parts[0:2]...),
			Host:   req.Host,
			Method: req.Method,
			Path:   req.URL.Path,
		}, nil
	}

	return &resolver.Endpoint{
		Name:   r.withNamespace(req, parts[0]),
		Host:   req.Host,
		Method: req.Method,
		Path:   req.URL.Path,
	}, nil
}

func (r *Resolver) String() string {
	return "path"
}

func (r *Resolver) withNamespace(req *http.Request, parts ...string) string {
	ns := r.opts.Namespace(req)
	if len(ns) == 0 {
		return strings.Join(parts, ".")
	}

	return strings.Join(append([]string{ns}, parts...), ".")
}
