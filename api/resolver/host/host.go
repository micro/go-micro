// Package host resolves using http host
package host

import (
	"net/http"

	"github.com/micro/go-micro/v3/api/resolver"
)

type Resolver struct {
	opts resolver.Options
}

func (r *Resolver) Resolve(req *http.Request, opts ...resolver.ResolveOption) (*resolver.Endpoint, error) {
	// parse options
	options := resolver.NewResolveOptions(opts...)

	return &resolver.Endpoint{
		Name:   req.Host,
		Host:   req.Host,
		Method: req.Method,
		Path:   req.URL.Path,
		Domain: options.Domain,
	}, nil
}

func (r *Resolver) String() string {
	return "host"
}

func NewResolver(opts ...resolver.Option) resolver.Resolver {
	return &Resolver{opts: resolver.NewOptions(opts...)}
}
