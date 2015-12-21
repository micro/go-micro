package gomicro

import (
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-micro/transport"
)

type Options struct {
	Broker    broker.Broker
	Client    client.Client
	Server    server.Server
	Registry  registry.Registry
	Transport transport.Transport
}

func newOptions(opts ...Option) Options {
	var opt Options
	for _, o := range opts {
		o(&opt)
	}

	if opt.Broker == nil {
		opt.Broker = broker.DefaultBroker
	}

	if opt.Client == nil {
		opt.Client = client.DefaultClient
	}

	if opt.Server == nil {
		opt.Server = server.DefaultServer
	}

	if opt.Registry == nil {
		opt.Registry = registry.DefaultRegistry
	}

	if opt.Transport == nil {
		opt.Transport = transport.DefaultTransport
	}

	return opt
}

func Broker(b broker.Broker) Option {
	return func(o *Options) {
		o.Broker = b
	}
}

func Client(c client.Client) Option {
	return func(o *Options) {
		o.Client = c
	}
}

func Server(s server.Server) Option {
	return func(o *Options) {
		o.Server = s
	}
}

func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

func Transport(t transport.Transport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}
