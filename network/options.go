package network

import (
	"github.com/micro/go-micro/network/resolver"
	"github.com/micro/go-micro/network/resolver/dns"
	"github.com/micro/go-micro/proxy"
	"github.com/micro/go-micro/proxy/mucp"
	"github.com/micro/go-micro/router"
	"github.com/micro/go-micro/tunnel"
)

type Option func(*Options)

// Options configure network
type Options struct {
	// Name of the network
	Name string
	// Address to bind to
	Address string
	// Tunnel is network tunnel
	Tunnel tunnel.Tunnel
	// Router is network router
	Router router.Router
	// Proxy is network proxy
	Proxy proxy.Proxy
	// Resolver is network resolver
	Resolver resolver.Resolver
}

// Name is the network name
func Name(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

// Address is the network address
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
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
		Name:     DefaultName,
		Address:  DefaultAddress,
		Tunnel:   tunnel.NewTunnel(),
		Router:   router.DefaultRouter,
		Proxy:    mucp.NewProxy(),
		Resolver: &dns.Resolver{},
	}
}
