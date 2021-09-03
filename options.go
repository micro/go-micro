package micro

import (
	"context"
	"time"

	"github.com/asim/go-micro/v3/auth"
	"github.com/asim/go-micro/v3/broker"
	"github.com/asim/go-micro/v3/cache"
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/config"
	"github.com/asim/go-micro/v3/debug/profile"
	"github.com/asim/go-micro/v3/debug/trace"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/runtime"
	"github.com/asim/go-micro/v3/selector"
	"github.com/asim/go-micro/v3/server"
	"github.com/asim/go-micro/v3/store"
	"github.com/asim/go-micro/v3/transport"
	"github.com/urfave/cli/v2"
)

// Options for micro service
type Options struct {
	Auth      auth.Auth
	Broker    broker.Broker
	Cache     cache.Cache
	Cmd       cmd.Cmd
	Config    config.Config
	Client    client.Client
	Server    server.Server
	Store     store.Store
	Registry  registry.Registry
	Runtime   runtime.Runtime
	Transport transport.Transport
	Profile   profile.Profile

	// Before and After funcs
	BeforeStart []func() error
	BeforeStop  []func() error
	AfterStart  []func() error
	AfterStop   []func() error

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context

	Signal bool
}

func newOptions(opts ...Option) Options {
	opt := Options{
		Auth:      auth.DefaultAuth,
		Broker:    broker.DefaultBroker,
		Cache:     cache.DefaultCache,
		Cmd:       cmd.DefaultCmd,
		Config:    config.DefaultConfig,
		Client:    client.DefaultClient,
		Server:    server.DefaultServer,
		Store:     store.DefaultStore,
		Registry:  registry.DefaultRegistry,
		Runtime:   runtime.DefaultRuntime,
		Transport: transport.DefaultTransport,
		Context:   context.Background(),
		Signal:    true,
	}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// Broker to be used for service
func Broker(b broker.Broker) Option {
	return func(o *Options) {
		o.Broker = b
		// Update Client and Server
		o.Client.Init(client.Broker(b))
		o.Server.Init(server.Broker(b))
	}
}

func Cache(c cache.Cache) Option {
	return func(o *Options) {
		o.Cache = c
	}
}

func Cmd(c cmd.Cmd) Option {
	return func(o *Options) {
		o.Cmd = c
	}
}

// Client to be used for service
func Client(c client.Client) Option {
	return func(o *Options) {
		o.Client = c
	}
}

// Context specifies a context for the service.
// Can be used to signal shutdown of the service and for extra option values.
func Context(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

// HandleSignal toggles automatic installation of the signal handler that
// traps TERM, INT, and QUIT.  Users of this feature to disable the signal
// handler, should control liveness of the service through the context.
func HandleSignal(b bool) Option {
	return func(o *Options) {
		o.Signal = b
	}
}

// Profile to be used for debug profile
func Profile(p profile.Profile) Option {
	return func(o *Options) {
		o.Profile = p
	}
}

// Server to be used for service
func Server(s server.Server) Option {
	return func(o *Options) {
		o.Server = s
	}
}

// Store sets the store to use
func Store(s store.Store) Option {
	return func(o *Options) {
		o.Store = s
	}
}

// Registry sets the registry for the service
// and the underlying components
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
		// Update Client and Server
		o.Client.Init(client.Registry(r))
		o.Server.Init(server.Registry(r))
		// Update Broker
		o.Broker.Init(broker.Registry(r))
	}
}

// Tracer sets the tracer for the service
func Tracer(t trace.Tracer) Option {
	return func(o *Options) {
		o.Server.Init(server.Tracer(t))
	}
}

// Auth sets the auth for the service
func Auth(a auth.Auth) Option {
	return func(o *Options) {
		o.Auth = a
	}
}

// Config sets the config for the service
func Config(c config.Config) Option {
	return func(o *Options) {
		o.Config = c
	}
}

// Selector sets the selector for the service client
func Selector(s selector.Selector) Option {
	return func(o *Options) {
		o.Client.Init(client.Selector(s))
	}
}

// Transport sets the transport for the service
// and the underlying components
func Transport(t transport.Transport) Option {
	return func(o *Options) {
		o.Transport = t
		// Update Client and Server
		o.Client.Init(client.Transport(t))
		o.Server.Init(server.Transport(t))
	}
}

// Runtime sets the runtime
func Runtime(r runtime.Runtime) Option {
	return func(o *Options) {
		o.Runtime = r
	}
}

// Convenience options

// Address sets the address of the server
func Address(addr string) Option {
	return func(o *Options) {
		o.Server.Init(server.Address(addr))
	}
}

// Name of the service
func Name(n string) Option {
	return func(o *Options) {
		o.Server.Init(server.Name(n))
	}
}

// Version of the service
func Version(v string) Option {
	return func(o *Options) {
		o.Server.Init(server.Version(v))
	}
}

// Metadata associated with the service
func Metadata(md map[string]string) Option {
	return func(o *Options) {
		o.Server.Init(server.Metadata(md))
	}
}

// Flags that can be passed to service
func Flags(flags ...cli.Flag) Option {
	return func(o *Options) {
		o.Cmd.App().Flags = append(o.Cmd.App().Flags, flags...)
	}
}

// Action can be used to parse user provided cli options
func Action(a func(*cli.Context) error) Option {
	return func(o *Options) {
		o.Cmd.App().Action = a
	}
}

// RegisterTTL specifies the TTL to use when registering the service
func RegisterTTL(t time.Duration) Option {
	return func(o *Options) {
		o.Server.Init(server.RegisterTTL(t))
	}
}

// RegisterInterval specifies the interval on which to re-register
func RegisterInterval(t time.Duration) Option {
	return func(o *Options) {
		o.Server.Init(server.RegisterInterval(t))
	}
}

// WrapClient is a convenience method for wrapping a Client with
// some middleware component. A list of wrappers can be provided.
// Wrappers are applied in reverse order so the last is executed first.
func WrapClient(w ...client.Wrapper) Option {
	return func(o *Options) {
		// apply in reverse
		for i := len(w); i > 0; i-- {
			o.Client = w[i-1](o.Client)
		}
	}
}

// WrapCall is a convenience method for wrapping a Client CallFunc
func WrapCall(w ...client.CallWrapper) Option {
	return func(o *Options) {
		o.Client.Init(client.WrapCall(w...))
	}
}

// WrapHandler adds a handler Wrapper to a list of options passed into the server
func WrapHandler(w ...server.HandlerWrapper) Option {
	return func(o *Options) {
		var wrappers []server.Option

		for _, wrap := range w {
			wrappers = append(wrappers, server.WrapHandler(wrap))
		}

		// Init once
		o.Server.Init(wrappers...)
	}
}

// WrapSubscriber adds a subscriber Wrapper to a list of options passed into the server
func WrapSubscriber(w ...server.SubscriberWrapper) Option {
	return func(o *Options) {
		var wrappers []server.Option

		for _, wrap := range w {
			wrappers = append(wrappers, server.WrapSubscriber(wrap))
		}

		// Init once
		o.Server.Init(wrappers...)
	}
}

// Before and Afters

// BeforeStart run funcs before service starts
func BeforeStart(fn func() error) Option {
	return func(o *Options) {
		o.BeforeStart = append(o.BeforeStart, fn)
	}
}

// BeforeStop run funcs before service stops
func BeforeStop(fn func() error) Option {
	return func(o *Options) {
		o.BeforeStop = append(o.BeforeStop, fn)
	}
}

// AfterStart run funcs after service starts
func AfterStart(fn func() error) Option {
	return func(o *Options) {
		o.AfterStart = append(o.AfterStart, fn)
	}
}

// AfterStop run funcs after service stops
func AfterStop(fn func() error) Option {
	return func(o *Options) {
		o.AfterStop = append(o.AfterStop, fn)
	}
}
