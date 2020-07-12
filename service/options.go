package service

import (
	"context"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/model"
	"github.com/micro/go-micro/v2/server"
)

type Options struct {
	Client client.Client
	Server server.Server
	Model  model.Model

	// Before and After funcs
	BeforeStart []func() error
	BeforeStop  []func() error
	AfterStart  []func() error
	AfterStop   []func() error

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type Option func(*Options)

func NewOptions(opts ...Option) Options {
	opt := Options{
		Client:  client.DefaultClient,
		Server:  server.DefaultServer,
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

func Client(c client.Client) Option {
	return func(o *Options) {
		o.Client = c
	}
}

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


// Context specifies a context for the service.
// Can be used to signal shutdown of the service.
// Can be used for extra option values.
func Context(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

// Server sets the server for handling requests
func Server(s server.Server) Option {
	return func(o *Options) {
		o.Server = s
	}
}

// Model sets the model for data access
func Model(m model.Model) Option {
	return func(o *Options) {
		o.Model = m
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

func BeforeStart(fn func() error) Option {
	return func(o *Options) {
		o.BeforeStart = append(o.BeforeStart, fn)
	}
}

func BeforeStop(fn func() error) Option {
	return func(o *Options) {
		o.BeforeStop = append(o.BeforeStop, fn)
	}
}

func AfterStart(fn func() error) Option {
	return func(o *Options) {
		o.AfterStart = append(o.AfterStart, fn)
	}
}

func AfterStop(fn func() error) Option {
	return func(o *Options) {
		o.AfterStop = append(o.AfterStop, fn)
	}
}
