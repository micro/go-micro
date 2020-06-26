// Package vpath resolves using http path and recognised versioned urls
package vpath

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/micro/go-micro/v2/api/resolver"
)

func NewResolver(opts ...resolver.Option) resolver.Resolver {
	return &Resolver{opts: resolver.NewOptions(opts...)}
}

type Resolver struct {
	opts resolver.Options
}

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
			Name:   r.withPrefix(parts...),
			Host:   req.Host,
			Method: req.Method,
			Path:   req.URL.Path,
			Domain: r.opts.Domain,
		}, nil
	}

	// /v1/foo
	if re.MatchString(parts[0]) {
		return &resolver.Endpoint{
			Name:   r.withPrefix(parts[0:2]...),
			Host:   req.Host,
			Method: req.Method,
			Path:   req.URL.Path,
			Domain: r.opts.Domain,
		}, nil
	}

	return &resolver.Endpoint{
		Name:   r.withPrefix(parts[0]),
		Host:   req.Host,
		Method: req.Method,
		Path:   req.URL.Path,
		Domain: r.opts.Domain,
	}, nil
}

func (r *Resolver) String() string {
	return "path"
}

// withPrefix transforms "foo" into "go.micro.api.foo"
func (r *Resolver) withPrefix(parts ...string) string {
	p := r.opts.ServicePrefix
	if len(p) > 0 {
		parts = append([]string{p}, parts...)
	}

	return strings.Join(parts, ".")
}
