// Package api is for building api gateways
package api

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"go-micro.dev/v4/api/router"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/server"
)

// API interface provides a way to
// create composable API gateways.
type Api interface {
	// Initialize options
	Init(...Option) error
	// Get the options
	Options() Options
	// Register an endpoint
	Register(*Endpoint) error
	// Deregister an endpoint
	Deregister(*Endpoint) error
	// Run the api
	Run(context.Context) error
	// Implemenation of api e.g http
	String() string
}

// Options are API options.
type Options struct {
	// Address of the server
	Address string
	// Router for resolving routes
	Router router.Router
	// Client to use for RPC
	Client client.Client
}

// Option type are API option args.
type Option func(*Options) error

// Endpoint is a mapping between an RPC method and HTTP endpoint.
type Endpoint struct {
	// RPC Method e.g. Greeter.Hello
	Name string
	// Description e.g what's this endpoint for
	Description string
	// API Handler e.g rpc, proxy
	Handler string
	// HTTP Host e.g example.com
	Host []string
	// HTTP Methods e.g GET, POST
	Method []string
	// HTTP Path e.g /greeter. Expect POSIX regex
	Path []string
	// Stream flag
	Stream bool
}

// Service represents an API service.
type Service struct {
	// Name of service
	Name string
	// The endpoint for this service
	Endpoint *Endpoint
	// Versions of this service
	Versions []*registry.Service
}

func strip(s string) string {
	return strings.TrimSpace(s)
}

func slice(s string) []string {
	var sl []string

	for _, p := range strings.Split(s, ",") {
		if str := strip(p); len(str) > 0 {
			sl = append(sl, strip(p))
		}
	}

	return sl
}

// Encode encodes an endpoint to endpoint metadata.
func Encode(e *Endpoint) map[string]string {
	if e == nil {
		return nil
	}

	// endpoint map
	em := make(map[string]string)

	// set vals only if they exist
	set := func(k, v string) {
		if len(v) == 0 {
			return
		}

		em[k] = v
	}

	set("endpoint", e.Name)
	set("description", e.Description)
	set("handler", e.Handler)
	set("method", strings.Join(e.Method, ","))
	set("path", strings.Join(e.Path, ","))
	set("host", strings.Join(e.Host, ","))

	return em
}

// Decode decodes endpoint metadata into an endpoint.
func Decode(e map[string]string) *Endpoint {
	if e == nil {
		return nil
	}

	return &Endpoint{
		Name:        e["endpoint"],
		Description: e["description"],
		Method:      slice(e["method"]),
		Path:        slice(e["path"]),
		Host:        slice(e["host"]),
		Handler:     e["handler"],
	}
}

// Validate validates an endpoint to guarantee it won't blow up when being served.
func Validate(e *Endpoint) error {
	if e == nil {
		return errors.New("endpoint is nil")
	}

	if len(e.Name) == 0 {
		return errors.New("name required")
	}

	for _, p := range e.Path {
		ps := p[0]
		pe := p[len(p)-1]

		if ps == '^' && pe == '$' {
			_, err := regexp.CompilePOSIX(p)
			if err != nil {
				return err
			}
		} else if ps == '^' && pe != '$' {
			return errors.New("invalid path")
		} else if ps != '^' && pe == '$' {
			return errors.New("invalid path")
		}
	}

	if len(e.Handler) == 0 {
		return errors.New("invalid handler")
	}

	return nil
}

// WithEndpoint returns a server.HandlerOption with endpoint metadata set
//
// Usage:
//
//	proto.RegisterHandler(service.Server(), new(Handler), api.WithEndpoint(
//		&api.Endpoint{
//			Name: "Greeter.Hello",
//			Path: []string{"/greeter"},
//		},
//	))
func WithEndpoint(e *Endpoint) server.HandlerOption {
	return server.EndpointMetadata(e.Name, Encode(e))
}

// NewApi returns a new api gateway.
func NewApi(opts ...Option) Api {
	return newApi(opts...)
}
