// Package proxy is a transparent proxy built on the go-micro/server
package proxy

import (
	"context"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/router"
	"github.com/micro/go-micro/server"
)

// Proxy can be used as a proxy server for go-micro services
type Proxy interface {
	options.Options
	// SendRequest honours the client.Router interface
	SendRequest(context.Context, client.Request, client.Response) error
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

// WithLink sets a link for outbound requests
func WithLink(name string, c client.Client) options.Option {
	return func(o *options.Values) error {
		var links map[string]client.Client
		v, ok := o.Get("proxy.links")
		if ok {
			links = v.(map[string]client.Client)
		} else {
			links = map[string]client.Client{}
		}
		links[name] = c
		// save the links
		o.Set("proxy.links", links)
		return nil
	}
}
