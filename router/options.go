package router

import (
	"context"

	"github.com/asim/nitro/v3/registry"
	"github.com/asim/nitro/v3/registry/memory"
	"github.com/google/uuid"
)

// Options are router options
type Options struct {
	// Id is router id
	Id string
	// Address is router address
	Address string
	// Gateway is network gateway
	Gateway string
	// Network is network address
	Network string
	// Registry is the local registry
	Registry registry.Registry
	// Context for additional options
	Context context.Context
	// Cache routes
	Cache bool
}

// Id sets Router Id
func Id(id string) Option {
	return func(o *Options) {
		o.Id = id
	}
}

// Address sets router service address
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}

// Gateway sets network gateway
func Gateway(g string) Option {
	return func(o *Options) {
		o.Gateway = g
	}
}

// Network sets router network
func Network(n string) Option {
	return func(o *Options) {
		o.Network = n
	}
}

// Registry sets the local registry
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// Cache the routes
func Cache() Option {
	return func(o *Options) {
		o.Cache = true
	}
}

// DefaultOptions returns router default options
func DefaultOptions() Options {
	return Options{
		Id:       uuid.New().String(),
		Network:  DefaultNetwork,
		Registry: memory.NewRegistry(),
		Context:  context.Background(),
	}
}

type ReadOptions struct {
	Service string
}

type ReadOption func(o *ReadOptions)

// ReadService sets the service to read from the table
func ReadService(s string) ReadOption {
	return func(o *ReadOptions) {
		o.Service = s
	}
}
