package web

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5"
	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/registry"
)

// Options for web.
type Options struct {
	Handler http.Handler

	Logger logger.Logger

	Service micro.Service

	Registry registry.Registry

	// Alternative Options
	Context context.Context

	Action    func(*cli.Context)
	Metadata  map[string]string
	TLSConfig *tls.Config

	Server *http.Server

	// RegisterCheck runs a check function before registering the service
	RegisterCheck func(context.Context) error

	Version string

	// Static directory
	StaticDir string

	Advertise string

	Address string
	Name    string
	Id      string
	Flags   []cli.Flag

	BeforeStart []func() error
	BeforeStop  []func() error
	AfterStart  []func() error
	AfterStop   []func() error

	RegisterInterval time.Duration

	RegisterTTL time.Duration

	Secure bool

	Signal bool
}

func newOptions(opts ...Option) Options {
	opt := Options{
		Name:             DefaultName,
		Version:          DefaultVersion,
		Id:               DefaultId,
		Address:          DefaultAddress,
		RegisterTTL:      DefaultRegisterTTL,
		RegisterInterval: DefaultRegisterInterval,
		StaticDir:        DefaultStaticDir,
		Service:          micro.NewService(),
		Context:          context.TODO(),
		Signal:           true,
		Logger:           logger.DefaultLogger,
	}

	for _, o := range opts {
		o(&opt)
	}

	if opt.RegisterCheck == nil {
		opt.RegisterCheck = DefaultRegisterCheck
	}

	return opt
}

// Name of Web.
func Name(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

// Icon specifies an icon url to load in the UI.
func Icon(ico string) Option {
	return func(o *Options) {
		if o.Metadata == nil {
			o.Metadata = make(map[string]string)
		}

		o.Metadata["icon"] = ico
	}
}

// Id for Unique server id.
func Id(id string) Option {
	return func(o *Options) {
		o.Id = id
	}
}

// Version of the service.
func Version(v string) Option {
	return func(o *Options) {
		o.Version = v
	}
}

// Metadata associated with the service.
func Metadata(md map[string]string) Option {
	return func(o *Options) {
		o.Metadata = md
	}
}

// Address to bind to - host:port.
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}

// Advertise The address to advertise for discovery - host:port.
func Advertise(a string) Option {
	return func(o *Options) {
		o.Advertise = a
	}
}

// Context specifies a context for the service.
// Can be used to signal shutdown of the service.
// Can be used for extra option values.
func Context(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

// Registry used for discovery.
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// RegisterTTL Register the service with a TTL.
func RegisterTTL(t time.Duration) Option {
	return func(o *Options) {
		o.RegisterTTL = t
	}
}

// RegisterInterval Register the service with at interval.
func RegisterInterval(t time.Duration) Option {
	return func(o *Options) {
		o.RegisterInterval = t
	}
}

// Handler for custom handler.
func Handler(h http.Handler) Option {
	return func(o *Options) {
		o.Handler = h
	}
}

// Server for custom Server.
func Server(srv *http.Server) Option {
	return func(o *Options) {
		o.Server = srv
	}
}

// MicroService sets the micro.Service used internally.
func MicroService(s micro.Service) Option {
	return func(o *Options) {
		o.Service = s
	}
}

// Flags sets the command flags.
func Flags(flags ...cli.Flag) Option {
	return func(o *Options) {
		o.Flags = append(o.Flags, flags...)
	}
}

// Action sets the command action.
func Action(a func(*cli.Context)) Option {
	return func(o *Options) {
		o.Action = a
	}
}

// BeforeStart is executed before the server starts.
func BeforeStart(fn func() error) Option {
	return func(o *Options) {
		o.BeforeStart = append(o.BeforeStart, fn)
	}
}

// BeforeStop is executed before the server stops.
func BeforeStop(fn func() error) Option {
	return func(o *Options) {
		o.BeforeStop = append(o.BeforeStop, fn)
	}
}

// AfterStart is executed after server start.
func AfterStart(fn func() error) Option {
	return func(o *Options) {
		o.AfterStart = append(o.AfterStart, fn)
	}
}

// AfterStop is executed after server stop.
func AfterStop(fn func() error) Option {
	return func(o *Options) {
		o.AfterStop = append(o.AfterStop, fn)
	}
}

// Secure Use secure communication.
// If TLSConfig is not specified we use InsecureSkipVerify and generate a self signed cert.
func Secure(b bool) Option {
	return func(o *Options) {
		o.Secure = b
	}
}

// TLSConfig to be used for the transport.
func TLSConfig(t *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = t
	}
}

// StaticDir sets the static file directory. This defaults to ./html.
func StaticDir(d string) Option {
	return func(o *Options) {
		o.StaticDir = d
	}
}

// RegisterCheck run func before registry service.
func RegisterCheck(fn func(context.Context) error) Option {
	return func(o *Options) {
		o.RegisterCheck = fn
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

// Logger sets the underline logger.
func Logger(l logger.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}
