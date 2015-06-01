package server

import (
	"github.com/myodc/go-micro/registry"
	"github.com/myodc/go-micro/transport"
)

type options struct {
	registry  registry.Registry
	transport transport.Transport
	metadata  map[string]string
	name      string
	address   string
	id        string
}

func newOptions(opt ...Option) options {
	var opts options

	for _, o := range opt {
		o(&opts)
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

	return opts
}

func (o options) Name() string {
	return o.name
}

func (o options) Id() string {
	return o.name + "-" + o.id
}

func (o options) Address() string {
	return o.address
}

func (o options) Metadata() map[string]string {
	return o.metadata
}
