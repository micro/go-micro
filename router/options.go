package router

import (
	"context"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v3/registry"
	"github.com/micro/go-micro/v3/registry/mdns"
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
	// Precache routes
	Precache bool
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

// Precache the routes
func Precache() Option {
	return func(o *Options) {
		o.Precache = true
	}
}

// DefaultOptions returns router default options
func DefaultOptions() Options {
	return Options{
		Id:       uuid.New().String(),
		Network:  DefaultNetwork,
		Registry: mdns.NewRegistry(),
		Context:  context.Background(),
	}
}
