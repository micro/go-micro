// Package host resolves using http host
package host

import (
	"net/http"

	"github.com/micro/go-micro/api/resolver"
)

type Resolver struct{}

func (r *Resolver) Resolve(req *http.Request) (*resolver.Endpoint, error) {
	return &resolver.Endpoint{
		Name:   req.Host,
		Host:   req.Host,
		Method: req.Method,
		Path:   req.URL.Path,
	}, nil
}

func (r *Resolver) String() string {
	return "host"
}

func NewResolver(opts ...resolver.Option) resolver.Resolver {
	return &Resolver{}
}
