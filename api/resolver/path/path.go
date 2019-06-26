// Package path resolves using http path
package path

import (
	"errors"
	"net/http"
	"strings"

	"github.com/micro/go-micro/api/resolver"
)

type Resolver struct{}

func (r *Resolver) Resolve(req *http.Request) (*resolver.Endpoint, error) {
	if req.URL.Path == "/" {
		return nil, errors.New("unknown name")
	}
	parts := strings.Split(req.URL.Path[1:], "/")
	return &resolver.Endpoint{
		Name:   parts[0],
		Host:   req.Host,
		Method: req.Method,
		Path:   req.URL.Path,
	}, nil
}

func (r *Resolver) String() string {
	return "path"
}

func NewResolver(opts ...resolver.Option) resolver.Resolver {
	return &Resolver{}
}
