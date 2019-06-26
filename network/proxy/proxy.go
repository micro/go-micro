// Package proxy is a transparent proxy built on the go-micro/server
package proxy

import (
	"context"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/network/router"
	"github.com/micro/go-micro/server"
)

// Proxy can be used as a proxy server for go-micro services
type Proxy interface {
	options.Options
	// ServeRequest honours the server.Router interface
	ServeRequest(context.Context, server.Request, server.Response) error
}

var (
	DefaultEndpoint = "localhost:9090"
)

// WithEndpoint sets a proxy endpoint
func WithEndpoint(e string) options.Option {
	return options.WithValue("proxy.endpoint", e)
}

// WithClient sets the client
func WithClient(c client.Client) options.Option {
	return options.WithValue("proxy.client", c)
}

// WithRouter specifies the router to use
func WithRouter(r router.Router) options.Option {
	return options.WithValue("proxy.router", r)
}
