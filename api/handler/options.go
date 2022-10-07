package handler

import (
	"go-micro.dev/v4/api/router"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/logger"
)

var (
	// DefaultMaxRecvSize is 10MiB.
	DefaultMaxRecvSize int64 = 1024 * 1024 * 100
)

// Options is the list of api Options.
type Options struct {
	MaxRecvSize int64
	Namespace   string
	Router      router.Router
	Client      client.Client
	Logger      logger.Logger
}

// Option is a api Option.
type Option func(o *Options)

// NewOptions fills in the blanks.
func NewOptions(opts ...Option) Options {
	options := Options{
		Logger: logger.DefaultLogger,
	}

	for _, o := range opts {
		o(&options)
	}

	if options.Client == nil {
		WithClient(client.DefaultClient)(&options)
	}

	if options.MaxRecvSize == 0 {
		options.MaxRecvSize = DefaultMaxRecvSize
	}

	if options.Logger == nil {
		options.Logger = logger.LoggerOrDefault(options.Logger)
	}

	return options
}

// WithNamespace specifies the namespace for the handler.
func WithNamespace(s string) Option {
	return func(o *Options) {
		o.Namespace = s
	}
}

// WithRouter specifies a router to be used by the handler.
func WithRouter(r router.Router) Option {
	return func(o *Options) {
		o.Router = r
	}
}

// WithClient sets the client for the handler.
func WithClient(c client.Client) Option {
	return func(o *Options) {
		o.Client = c
	}
}

// WithMaxRecvSize specifies max body size.
func WithMaxRecvSize(size int64) Option {
	return func(o *Options) {
		o.MaxRecvSize = size
	}
}

// WithLogger specifies the logger.
func WithLogger(l logger.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}
