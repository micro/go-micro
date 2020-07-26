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
	// Advertise is the advertising strategy
	Advertise Strategy
	// Context for additional options
	Context context.Context
	// Precache the route table on router startup
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

// Advertise sets route advertising strategy
func Advertise(a Strategy) Option {
	return func(o *Options) {
		o.Advertise = a
	}
}

// Precache sets whether to precache the route table
func Precache(b bool) Option {
	return func(o *Options) {
		o.Precache = b
	}
}

// DefaultOptions returns router default options
func DefaultOptions() Options {
	return Options{
		Id:        uuid.New().String(),
		Address:   DefaultAddress,
		Network:   DefaultNetwork,
		Registry:  mdns.NewRegistry(),
		Advertise: AdvertiseLocal,
		Context:   context.Background(),
	}
}
