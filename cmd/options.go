package cmd

import (
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/selector"
	"github.com/micro/go-micro/transport"
)

type Options struct {
	Name        string
	Description string
	Version     string

	Brokers    map[string]func([]string, ...broker.Option) broker.Broker
	Registries map[string]func([]string, ...registry.Option) registry.Registry
	Selectors  map[string]func(...selector.Option) selector.Selector
	Transports map[string]func([]string, ...transport.Option) transport.Transport
}

func Name(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

func Description(d string) Option {
	return func(o *Options) {
		o.Description = d
	}
}

func Version(v string) Option {
	return func(o *Options) {
		o.Version = v
	}
}

func Broker(name string, b func([]string, ...broker.Option) broker.Broker) Option {
	return func(o *Options) {
		o.Brokers[name] = b
	}
}

func Registry(name string, r func([]string, ...registry.Option) registry.Registry) Option {
	return func(o *Options) {
		o.Registries[name] = r
	}
}

func Selector(name string, s func(...selector.Option) selector.Selector) Option {
	return func(o *Options) {
		o.Selectors[name] = s
	}
}

func Transport(name string, t func([]string, ...transport.Option) transport.Transport) Option {
	return func(o *Options) {
		o.Transports[name] = t
	}
}
