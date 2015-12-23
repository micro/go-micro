package server

import (
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/transport"
)

type options struct {
	codecs       map[string]codec.NewCodec
	broker       broker.Broker
	registry     registry.Registry
	transport    transport.Transport
	metadata     map[string]string
	name         string
	address      string
	advertise    string
	id           string
	version      string
	hdlrWrappers []HandlerWrapper
	subWrappers  []SubscriberWrapper
}

func newOptions(opt ...Option) options {
	opts := options{
		codecs:   make(map[string]codec.NewCodec),
		metadata: map[string]string{},
	}

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

// Server name
func Name(n string) Option {
	return func(o *options) {
		o.name = n
	}
}

// Unique server id
func Id(id string) Option {
	return func(o *options) {
		o.id = id
	}
}

// Version of the service
func Version(v string) Option {
	return func(o *options) {
		o.version = v
	}
}

// Address to bind to - host:port
func Address(a string) Option {
	return func(o *options) {
		o.address = a
	}
}

// The address to advertise for discovery - host:port
func Advertise(a string) Option {
	return func(o *options) {
		o.advertise = a
	}
}

// Broker to use for pub/sub
func Broker(b broker.Broker) Option {
	return func(o *options) {
		o.broker = b
	}
}

// Codec to use to encode/decode requests for a given content type
func Codec(contentType string, c codec.NewCodec) Option {
	return func(o *options) {
		o.codecs[contentType] = c
	}
}

// Registry used for discovery
func Registry(r registry.Registry) Option {
	return func(o *options) {
		o.registry = r
	}
}

// Transport mechanism for communication e.g http, rabbitmq, etc
func Transport(t transport.Transport) Option {
	return func(o *options) {
		o.transport = t
	}
}

// Metadata associated with the server
func Metadata(md map[string]string) Option {
	return func(o *options) {
		o.metadata = md
	}
}

// Adds a handler Wrapper to a list of options passed into the server
func WrapHandler(w HandlerWrapper) Option {
	return func(o *options) {
		o.hdlrWrappers = append(o.hdlrWrappers, w)
	}
}

// Adds a subscriber Wrapper to a list of options passed into the server
func WrapSubscriber(w SubscriberWrapper) Option {
	return func(o *options) {
		o.subWrappers = append(o.subWrappers, w)
	}
}
