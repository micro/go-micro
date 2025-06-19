package cmd

import (
	"context"

	"go-micro.dev/v5/auth"
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/cache"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/config"
	"go-micro.dev/v5/debug/profile"
	"go-micro.dev/v5/debug/trace"
	"go-micro.dev/v5/events"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/selector"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/store"
	"go-micro.dev/v5/transport"
)

type Options struct {

	// Other options for implementations of the interface
	// can be stored in a context
	Context      context.Context
	Auth         *auth.Auth
	Selector     *selector.Selector
	DebugProfile *profile.Profile

	Registry *registry.Registry

	Brokers       map[string]func(...broker.Option) broker.Broker
	Transport     *transport.Transport
	Cache         *cache.Cache
	Config        *config.Config
	Client        *client.Client
	Server        *server.Server
	Caches        map[string]func(...cache.Option) cache.Cache
	Tracer        *trace.Tracer
	DebugProfiles map[string]func(...profile.Option) profile.Profile

	// We need pointers to things so we can swap them out if needed.
	Broker     *broker.Broker
	Auths      map[string]func(...auth.Option) auth.Auth
	Store      *store.Store
	Stream     *events.Stream
	Configs    map[string]func(...config.Option) (config.Config, error)
	Clients    map[string]func(...client.Option) client.Client
	Registries map[string]func(...registry.Option) registry.Registry
	Selectors  map[string]func(...selector.Option) selector.Selector
	Servers    map[string]func(...server.Option) server.Server
	Transports map[string]func(...transport.Option) transport.Transport
	Stores     map[string]func(...store.Option) store.Store
	Streams    map[string]func(...events.Option) events.Stream
	Tracers    map[string]func(...trace.Option) trace.Tracer
	Version    string

	// For the Command Line itself
	Name        string
	Description string
}

// Command line Name.
func Name(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

// Command line Description.
func Description(d string) Option {
	return func(o *Options) {
		o.Description = d
	}
}

// Command line Version.
func Version(v string) Option {
	return func(o *Options) {
		o.Version = v
	}
}

func Broker(b *broker.Broker) Option {
	return func(o *Options) {
		o.Broker = b
		broker.DefaultBroker = *b
	}
}

func Cache(c *cache.Cache) Option {
	return func(o *Options) {
		o.Cache = c
		cache.DefaultCache = *c
	}
}

func Config(c *config.Config) Option {
	return func(o *Options) {
		o.Config = c
		config.DefaultConfig = *c
	}
}

func Selector(s *selector.Selector) Option {
	return func(o *Options) {
		o.Selector = s
		selector.DefaultSelector = *s
	}
}

func Registry(r *registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
		registry.DefaultRegistry = *r
	}
}

func Transport(t *transport.Transport) Option {
	return func(o *Options) {
		o.Transport = t
		transport.DefaultTransport = *t
	}
}

func Client(c *client.Client) Option {
	return func(o *Options) {
		o.Client = c
		client.DefaultClient = *c
	}
}

func Server(s *server.Server) Option {
	return func(o *Options) {
		o.Server = s
		server.DefaultServer = *s
	}
}

func Store(s *store.Store) Option {
	return func(o *Options) {
		o.Store = s
		store.DefaultStore = *s
	}
}

func Stream(s *events.Stream) Option {
	return func(o *Options) {
		o.Stream = s
		events.DefaultStream = *s
	}
}

func Tracer(t *trace.Tracer) Option {
	return func(o *Options) {
		o.Tracer = t
		trace.DefaultTracer = *t
	}
}

func Auth(a *auth.Auth) Option {
	return func(o *Options) {
		o.Auth = a
		auth.DefaultAuth = *a
	}
}

func Profile(p *profile.Profile) Option {
	return func(o *Options) {
		o.DebugProfile = p
		profile.DefaultProfile = *p
	}
}

// New broker func.
func NewBroker(name string, b func(...broker.Option) broker.Broker) Option {
	return func(o *Options) {
		o.Brokers[name] = b
	}
}

// New stream func.
func NewStream(name string, b func(...events.Option) events.Stream) Option {
	return func(o *Options) {
		o.Streams[name] = b
	}
}

// New cache func.
func NewCache(name string, c func(...cache.Option) cache.Cache) Option {
	return func(o *Options) {
		o.Caches[name] = c
	}
}

// New client func.
func NewClient(name string, b func(...client.Option) client.Client) Option {
	return func(o *Options) {
		o.Clients[name] = b
	}
}

// New registry func.
func NewRegistry(name string, r func(...registry.Option) registry.Registry) Option {
	return func(o *Options) {
		o.Registries[name] = r
	}
}

// New selector func.
func NewSelector(name string, s func(...selector.Option) selector.Selector) Option {
	return func(o *Options) {
		o.Selectors[name] = s
	}
}

// New server func.
func NewServer(name string, s func(...server.Option) server.Server) Option {
	return func(o *Options) {
		o.Servers[name] = s
	}
}

// New transport func.
func NewTransport(name string, t func(...transport.Option) transport.Transport) Option {
	return func(o *Options) {
		o.Transports[name] = t
	}
}

// New tracer func.
func NewTracer(name string, t func(...trace.Option) trace.Tracer) Option {
	return func(o *Options) {
		o.Tracers[name] = t
	}
}

// New auth func.
func NewAuth(name string, t func(...auth.Option) auth.Auth) Option {
	return func(o *Options) {
		o.Auths[name] = t
	}
}

// New config func.
func NewConfig(name string, t func(...config.Option) (config.Config, error)) Option {
	return func(o *Options) {
		o.Configs[name] = t
	}
}

// New profile func.
func NewProfile(name string, t func(...profile.Option) profile.Profile) Option {
	return func(o *Options) {
		o.DebugProfiles[name] = t
	}
}
