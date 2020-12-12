// Package proxy is a transparent proxy built on the go-micro/server
package proxy

import (
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/router"
)

type Options struct {
	// Specific endpoint to always call
	Endpoint string
	// The default client to use
	Client client.Client
	// The default router to use
	Router router.Router
	// Extra links for different clients
	Links map[string]client.Client
}

type Option func(o *Options)

// WithEndpoint sets a proxy endpoint
func WithEndpoint(e string) Option {
	return func(o *Options) {
		o.Endpoint = e
	}
}

// WithClient sets the client
func WithClient(c client.Client) Option {
	return func(o *Options) {
		o.Client = c
	}
}

// WithRouter specifies the router to use
func WithRouter(r router.Router) Option {
	return func(o *Options) {
		o.Router = r
	}
}

// WithLink sets a link for outbound requests
func WithLink(name string, c client.Client) Option {
	return func(o *Options) {
		if o.Links == nil {
			o.Links = make(map[string]client.Client)
		}
		o.Links[name] = c
	}
}
