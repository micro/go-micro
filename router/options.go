package router

import (
	"github.com/google/uuid"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/registry"
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
	// Client for calling router
	Client client.Client
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

// Client to call router service
func Client(c client.Client) Option {
	return func(o *Options) {
		o.Client = c
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

// Strategy sets route advertising strategy
func Advertise(a Strategy) Option {
	return func(o *Options) {
		o.Advertise = a
	}
}

// DefaultOptions returns router default options
func DefaultOptions() Options {
	return Options{
		Id:        uuid.New().String(),
		Address:   DefaultAddress,
		Network:   DefaultNetwork,
		Registry:  registry.DefaultRegistry,
		Advertise: AdvertiseLocal,
	}
}
