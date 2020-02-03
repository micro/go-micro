package server

import (
	"crypto/tls"

	"github.com/micro/go-micro/v2/api/server/acme"
)

type Option func(o *Options)

type Options struct {
	EnableACME   bool
	ACMEProvider acme.Provider
	EnableTLS    bool
	ACMEHosts    []string
	TLSConfig    *tls.Config
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
