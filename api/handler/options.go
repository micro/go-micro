package handler

import (
	"github.com/micro/go-micro/v2/api/router"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/grpc"
)

var (
	DefaultMaxRecvSize int64 = 1024 * 1024 * 100 // 10Mb
)

type Options struct {
	MaxRecvSize int64
	Namespace   string
	Router      router.Router
	Client      client.Client
}

type Option func(o *Options)

// NewOptions fills in the blanks
func NewOptions(opts ...Option) Options {
	var options Options
	for _, o := range opts {
		o(&options)
	}

	if options.Client == nil {
		WithClient(grpc.NewClient())(&options)
	}

	// set namespace if blank
	if len(options.Namespace) == 0 {
		WithNamespace("go.micro.api")(&options)
	}

	if options.MaxRecvSize == 0 {
		options.MaxRecvSize = DefaultMaxRecvSize
	}

	return options
}

// WithNamespace specifies the namespace for the handler
func WithNamespace(s string) Option {
	return func(o *Options) {
		o.Namespace = s
	}
}

// WithRouter specifies a router to be used by the handler
func WithRouter(r router.Router) Option {
	return func(o *Options) {
		o.Router = r
	}
}

func WithClient(c client.Client) Option {
	return func(o *Options) {
		o.Client = c
	}
}

// WithmaxRecvSize specifies max body size
func WithMaxRecvSize(size int64) Option {
	return func(o *Options) {
		o.MaxRecvSize = size
	}
}
