package network

import (
	"github.com/micro/go-micro/network/resolver"
	"github.com/micro/go-micro/network/resolver/dns"
)

type Option func(*Options)

// Options configure network
type Options struct {
	// Name of the network
	Name string
	// Address to bind to
	Address string
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
		Resolver: &dns.Resolver{},
	}
}
