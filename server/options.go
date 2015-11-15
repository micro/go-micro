package server

import (
	"github.com/piemapping/go-micro/broker"
	"github.com/piemapping/go-micro/registry"
	"github.com/piemapping/go-micro/transport"
)

type options struct {
	broker    broker.Broker
	registry  registry.Registry
	transport transport.Transport
	metadata  map[string]string
	name      string
	address   string
	advertise string
	id        string
	version   string
}

func newOptions(opt ...Option) options {
	var opts options

	for _, o := range opt {
		o(&opts)
	}

	if opts.broker == nil {
		opts.broker = broker.DefaultBroker
	}

	if opts.registry == nil {
		opts.registry = registry.DefaultRegistry
	}

	if opts.transport == nil {
		opts.transport = transport.DefaultTransport
	}

	if len(opts.address) == 0 {
		opts.address = DefaultAddress
	}

	if len(opts.name) == 0 {
		opts.name = DefaultName
	}

	if len(opts.id) == 0 {
		opts.id = DefaultId
	}

	if len(opts.version) == 0 {
		opts.version = DefaultVersion
	}

	return opts
}

func (o options) Name() string {
	return o.name
}

func (o options) Id() string {
	return o.name + "-" + o.id
}

func (o options) Version() string {
	return o.version
}

func (o options) Address() string {
	return o.address
}

func (o options) Advertise() string {
	return o.advertise
}

func (o options) Metadata() map[string]string {
	return o.metadata
}

func Name(n string) Option {
	return func(o *options) {
		o.name = n
	}
}

func Id(id string) Option {
	return func(o *options) {
		o.id = id
	}
}

func Version(v string) Option {
	return func(o *options) {
		o.version = v
	}
}

func Address(a string) Option {
	return func(o *options) {
		o.address = a
	}
}

func Advertise(a string) Option {
	return func(o *options) {
		o.advertise = a
	}
}

func Broker(b broker.Broker) Option {
	return func(o *options) {
		o.broker = b
	}
}

func Registry(r registry.Registry) Option {
	return func(o *options) {
		o.registry = r
	}
}

func Transport(t transport.Transport) Option {
	return func(o *options) {
		o.transport = t
	}
}

func Metadata(md map[string]string) Option {
	return func(o *options) {
		o.metadata = md
	}
}
