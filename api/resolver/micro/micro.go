// Package micro provides a micro rpc resolver which prefixes a namespace
package micro

import (
	"net/http"

	"github.com/micro/go-micro/api/resolver"
)

// default resolver for legacy purposes
// it uses proxy routing to resolve names
// /foo becomes namespace.foo
// /v1/foo becomes namespace.v1.foo
type Resolver struct {
	Options resolver.Options
}

func (r *Resolver) Resolve(req *http.Request) (*resolver.Endpoint, error) {
	var name, method string

	switch r.Options.Handler {
	// internal handlers
	case "meta", "api", "rpc", "micro":
		name, method = apiRoute(req.URL.Path)
	default:
		method = req.Method
		name = proxyRoute(req.URL.Path)
	}

	return &resolver.Endpoint{
		Name:   name,
		Method: method,
	}, nil
}

func (r *Resolver) String() string {
	return "micro"
}

// NewResolver creates a new micro resolver
func NewResolver(opts ...resolver.Option) resolver.Resolver {
	return &Resolver{
		Options: resolver.NewOptions(opts...),
	}
}
