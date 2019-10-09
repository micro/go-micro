package server

import (
	"crypto/tls"

	"github.com/micro/go-micro/api/server/acme"
)

type Option func(o *Options)

type Options struct {
	EnableACME  bool
	ACMELibrary acme.Library
	EnableTLS   bool
	ACMEHosts   []string
	TLSConfig   *tls.Config
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

func ACMELibrary(lib acme.Library) Option {
	return func(o *Options) {
		o.ACMELibrary = lib
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
