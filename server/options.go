package server

import (
	"context"
	"time"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/server/debug"
	"github.com/micro/go-micro/transport"
)

type Options struct {
	Codecs       map[string]codec.NewCodec
	Broker       broker.Broker
	Registry     registry.Registry
	Transport    transport.Transport
	Metadata     map[string]string
	Name         string
	Address      string
	Advertise    string
	Id           string
	Version      string
	HdlrWrappers []HandlerWrapper
	SubWrappers  []SubscriberWrapper

	RegisterTTL time.Duration

	// Debug Handler which can be set by a user
	DebugHandler debug.DebugHandler

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

func newOptions(opt ...Option) Options {
	opts := Options{
		Codecs:   make(map[string]codec.NewCodec),
		Metadata: map[string]string{},
	}

	for _, o := range opt {
		o(&opts)
	}

	if opts.Broker == nil {
		opts.Broker = broker.DefaultBroker
	}

	if opts.Registry == nil {
		opts.Registry = registry.DefaultRegistry
	}

	if opts.Transport == nil {
		opts.Transport = transport.DefaultTransport
	}

	if opts.DebugHandler == nil {
		opts.DebugHandler = debug.DefaultDebugHandler
	}

	if len(opts.Address) == 0 {
		opts.Address = DefaultAddress
	}

	if len(opts.Name) == 0 {
		opts.Name = DefaultName
	}

	if len(opts.Id) == 0 {
		opts.Id = DefaultId
	}

	if len(opts.Version) == 0 {
		opts.Version = DefaultVersion
	}

	return opts
}

// Server name
func Name(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

// Unique server id
func Id(id string) Option {
	return func(o *Options) {
		o.Id = id
	}
}

// Version of the service
func Version(v string) Option {
	return func(o *Options) {
		o.Version = v
	}
}

// Address to bind to - host:port
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}

// The address to advertise for discovery - host:port
func Advertise(a string) Option {
	return func(o *Options) {
		o.Advertise = a
	}
}

// Broker to use for pub/sub
func Broker(b broker.Broker) Option {
	return func(o *Options) {
		o.Broker = b
	}
}

// Codec to use to encode/decode requests for a given content type
func Codec(contentType string, c codec.NewCodec) Option {
	return func(o *Options) {
		o.Codecs[contentType] = c
	}
}

// Registry used for discovery
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// Transport mechanism for communication e.g http, rabbitmq, etc
func Transport(t transport.Transport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}

// DebugHandler for this server
func DebugHandler(d debug.DebugHandler) Option {
	return func(o *Options) {
		o.DebugHandler = d
	}
}

// Metadata associated with the server
func Metadata(md map[string]string) Option {
	return func(o *Options) {
		o.Metadata = md
	}
}

// Register the service with a TTL
func RegisterTTL(t time.Duration) Option {
	return func(o *Options) {
		o.RegisterTTL = t
	}
}

// Wait tells the server to wait for requests to finish before exiting
func Wait(b bool) Option {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, "wait", b)
	}
}

// Adds a handler Wrapper to a list of options passed into the server
func WrapHandler(w HandlerWrapper) Option {
	return func(o *Options) {
		o.HdlrWrappers = append(o.HdlrWrappers, w)
	}
}

// Adds a subscriber Wrapper to a list of options passed into the server
func WrapSubscriber(w SubscriberWrapper) Option {
	return func(o *Options) {
		o.SubWrappers = append(o.SubWrappers, w)
	}
}
