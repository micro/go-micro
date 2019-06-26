package server

import (
	"crypto/tls"
)

type Option func(o *Options)

type Options struct {
	EnableACME bool
	EnableTLS  bool
	ACMEHosts  []string
	TLSConfig  *tls.Config
}

func ACMEHosts(hosts ...string) Option {
	return func(o *Options) {
		o.ACMEHosts = hosts
	}
}

func EnableACME(b bool) Option {
	return func(o *Options) {
		o.EnableACME = b
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
