package api

import (
	"crypto/tls"

	"github.com/asim/go-micro/v3/acme"
	"github.com/asim/go-micro/v3/api/resolver"
)

type Options struct {
	Address      string
	EnableACME   bool
	ACMEProvider acme.Provider
	EnableTLS    bool
	ACMEHosts    []string
	TLSConfig    *tls.Config
	Resolver     resolver.Resolver
}

type Option func(o *Options)

func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}

func EnableACME(b bool) Option {
	return func(o *Options) {
		o.EnableACME = b
	}
}

func ACMEHosts(hosts ...string) Option {
	return func(o *Options) {
		o.ACMEHosts = hosts
	}
}

func ACMEProvider(p acme.Provider) Option {
	return func(o *Options) {
		o.ACMEProvider = p
	}
}

func EnableTLS(b bool) Option {
	return func(o *Options) {
		o.EnableTLS = b
	}
}

func TLSConfig(t *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = t
	}
}

func Resolver(r resolver.Resolver) Option {
	return func(o *Options) {
		o.Resolver = r
	}
}
