package network

import (
	"github.com/google/uuid"
	"github.com/micro/go-micro/v2/network/resolver"
	"github.com/micro/go-micro/v2/network/resolver/registry"
	"github.com/micro/go-micro/v2/proxy"
	"github.com/micro/go-micro/v2/proxy/mucp"
	"github.com/micro/go-micro/v2/router"
	"github.com/micro/go-micro/v2/tunnel"
)

type Option func(*Options)

// Options configure network
type Options struct {
	// Id of the node
	Id string
	// Name of the network
	Name string
	// Address to bind to
	Address string
	// Advertise sets the address to advertise
	Advertise string
	// Nodes is a list of nodes to connect to
	Nodes []string
	// Tunnel is network tunnel
	Tunnel tunnel.Tunnel
	// Router is network router
	Router router.Router
	// Proxy is network proxy
	Proxy proxy.Proxy
	// Resolver is network resolver
	Resolver resolver.Resolver
}

// Id sets the id of the network node
func Id(id string) Option {
	return func(o *Options) {
		o.Id = id
	}
}

// Name sets the network name
func Name(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

// Address sets the network address
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}

// Advertise sets the address to advertise
func Advertise(a string) Option {
	return func(o *Options) {
		o.Advertise = a
	}
}

// Nodes is a list of nodes to connect to
func Nodes(n ...string) Option {
	return func(o *Options) {
		o.Nodes = n
	}
}

// Tunnel sets the network tunnel
func Tunnel(t tunnel.Tunnel) Option {
	return func(o *Options) {
		o.Tunnel = t
	}
}

// Router sets the network router
func Router(r router.Router) Option {
	return func(o *Options) {
		o.Router = r
	}
}

// Proxy sets the network proxy
func Proxy(p proxy.Proxy) Option {
	return func(o *Options) {
		o.Proxy = p
	}
}

// Resolver is the network resolver
func Resolver(r resolver.Resolver) Option {
	return func(o *Options) {
		o.Resolver = r
	}
}

// DefaultOptions returns network default options
func DefaultOptions() Options {
	return Options{
		Id:       uuid.New().String(),
		Name:     DefaultName,
		Address:  DefaultAddress,
		Tunnel:   tunnel.NewTunnel(),
		Router:   router.DefaultRouter,
		Proxy:    mucp.NewProxy(),
		Resolver: &registry.Resolver{},
	}
}
