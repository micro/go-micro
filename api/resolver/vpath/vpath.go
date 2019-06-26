// Package vpath resolves using http path and recognised versioned urls
package vpath

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/micro/go-micro/api/resolver"
)

type Resolver struct{}

var (
	re = regexp.MustCompile("^v[0-9]+$")
)

func (r *Resolver) Resolve(req *http.Request) (*resolver.Endpoint, error) {
	if req.URL.Path == "/" {
		return nil, errors.New("unknown name")
	}

	parts := strings.Split(req.URL.Path[1:], "/")

	if len(parts) == 1 {
		return &resolver.Endpoint{
			Name:   parts[0],
			Host:   req.Host,
			Method: req.Method,
			Path:   req.URL.Path,
		}, nil
	}

	// /v1/foo
	if re.MatchString(parts[0]) {
		return &resolver.Endpoint{
			Name:   parts[1],
			Host:   req.Host,
			Method: req.Method,
			Path:   req.URL.Path,
		}, nil
	}

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
