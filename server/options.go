package server

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	"github.com/asim/go-micro/v3/broker"
	"github.com/asim/go-micro/v3/codec"
	"github.com/asim/go-micro/v3/debug/trace"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/transport"
)

type Options struct {
	Codecs       map[string]codec.NewCodec
	Broker       broker.Broker
	Registry     registry.Registry
	Tracer       trace.Tracer
	Transport    transport.Transport
	Metadata     map[string]string
	Name         string
	Address      string
	Advertise    string
	Id           string
	Version      string
	HdlrWrappers []HandlerWrapper
	SubWrappers  []SubscriberWrapper

	// RegisterCheck runs a check function before registering the service
	RegisterCheck func(context.Context) error
	// The register expiry time
	RegisterTTL time.Duration
	// The interval on which to register
	RegisterInterval time.Duration

	// The router for requests
	Router Router

	// TLSConfig specifies tls.Config for secure serving
	TLSConfig *tls.Config

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

func newOptions(opt ...Option) Options {
	opts := Options{
		Codecs:           make(map[string]codec.NewCodec),
		Metadata:         map[string]string{},
		RegisterInterval: DefaultRegisterInterval,
		RegisterTTL:      DefaultRegisterTTL,
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

	if opts.RegisterCheck == nil {
		opts.RegisterCheck = DefaultRegisterCheck
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

// Context specifies a context for the service.
// Can be used to signal shutdown of the service
// Can be used for extra option values.
func Context(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

// Registry used for discovery
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// Tracer mechanism for distributed tracking
func Tracer(t trace.Tracer) Option {
	return func(o *Options) {
		o.Tracer = t
	}
}

// Transport mechanism for communication e.g http, rabbitmq, etc
func Transport(t transport.Transport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}

// Metadata associated with the server
func Metadata(md map[string]string) Option {
	return func(o *Options) {
		o.Metadata = md
	}
}

// RegisterCheck run func before registry service
func RegisterCheck(fn func(context.Context) error) Option {
	return func(o *Options) {
		o.RegisterCheck = fn
	}
}

// Register the service with a TTL
func RegisterTTL(t time.Duration) Option {
	return func(o *Options) {
		o.RegisterTTL = t
	}
}

// Register the service with at interval
func RegisterInterval(t time.Duration) Option {
	return func(o *Options) {
		o.RegisterInterval = t
	}
}

// TLSConfig specifies a *tls.Config
func TLSConfig(t *tls.Config) Option {
	return func(o *Options) {
		// set the internal tls
		o.TLSConfig = t

		// set the default transport if one is not
		// already set. Required for Init call below.
		if o.Transport == nil {
			o.Transport = transport.DefaultTransport
		}

		// set the transport tls
		o.Transport.Init(
			transport.Secure(true),
			transport.TLSConfig(t),
		)
	}
}

// WithRouter sets the request router
func WithRouter(r Router) Option {
	return func(o *Options) {
		o.Router = r
	}
}

// Wait tells the server to wait for requests to finish before exiting
// If `wg` is nil, server only wait for completion of rpc handler.
// For user need finer grained control, pass a concrete `wg` here, server will
// wait against it on stop.
func Wait(wg *sync.WaitGroup) Option {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		if wg == nil {
			wg = new(sync.WaitGroup)
		}
		o.Context = context.WithValue(o.Context, "wait", wg)
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
