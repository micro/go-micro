// Package grpc resolves a grpc service like /greeter.Say/Hello to greeter service
package grpc

import (
	"errors"
	"net/http"
	"strings"

	"github.com/micro/go-micro/v2/api/resolver"
)

type Resolver struct{}

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

func (r *Resolver) String() string {
	return "grpc"
}

func NewResolver(opts ...resolver.Option) resolver.Resolver {
	return &Resolver{}
}
