package api

import (
	"errors"
	"regexp"
	"strings"

	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/server"
)

// Endpoint is a mapping between an RPC method and HTTP endpoint
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
}

// Service represents an API service
type Service struct {
	// Name of service
	Name string
	// The endpoint for this service
	Endpoint *Endpoint
	// Versions of this service
	Services []*registry.Service
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

// Encode encodes an endpoint to endpoint metadata
func Encode(e *Endpoint) map[string]string {
	if e == nil {
		return nil
	}

	return map[string]string{
		"endpoint":    e.Name,
		"description": e.Description,
		"method":      strings.Join(e.Method, ","),
		"path":        strings.Join(e.Path, ","),
		"host":        strings.Join(e.Host, ","),
		"handler":     e.Handler,
	}
}

// Decode decodes endpoint metadata into an endpoint
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

// Validate validates an endpoint to guarantee it won't blow up when being served
func Validate(e *Endpoint) error {
	if e == nil {
		return errors.New("endpoint is nil")
	}

	if len(e.Name) == 0 {
		return errors.New("name required")
	}

	for _, p := range e.Path {
		_, err := regexp.CompilePOSIX(p)
		if err != nil {
			return err
		}
	}

	if len(e.Handler) == 0 {
		return errors.New("invalid handler")
	}

	return nil
}

/*
Design ideas

// Gateway is an api gateway interface
type Gateway interface {
	// Register a http handler
	Handle(pattern string, http.Handler)
	// Register a route
	RegisterRoute(r Route)
	// Init initialises the command line.
	// It also parses further options.
	Init(...Option) error
	// Run the gateway
	Run() error
}

// NewGateway returns a new api gateway
func NewGateway() Gateway {
	return newGateway()
}
*/

// WithEndpoint returns a server.HandlerOption with endpoint metadata set
//
// Usage:
//
// 	proto.RegisterHandler(service.Server(), new(Handler), api.WithEndpoint(
//		&api.Endpoint{
//			Name: "Greeter.Hello",
//			Path: []string{"/greeter"},
//		},
//	))
func WithEndpoint(e *Endpoint) server.HandlerOption {
	return server.EndpointMetadata(e.Name, Encode(e))
}
