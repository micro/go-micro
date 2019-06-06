package grpc

import (
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/server"
)

// WithBackend provides an option to set the proxy backend url
func WithBackend(url string) micro.Option {
	return func(o *micro.Options) {
		// get the router
		r := o.Server.Options().Router

		// not set
		if r == nil {
			r = DefaultProxy
			o.Server.Init(server.WithRouter(r))
		}

		// check its a proxy router
		if proxyRouter, ok := r.(*Proxy); ok {
			proxyRouter.Backend = url
		}
	}
}

// WithRouter provides an option to set the proxy router
func WithRouter(r server.Router) micro.Option {
	return func(o *micro.Options) {
		o.Server.Init(server.WithRouter(r))
	}
}
