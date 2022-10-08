package server

import (
	"crypto/tls"
	"net/http"

	"go-micro.dev/v4/api/resolver"
	"go-micro.dev/v4/api/server/acme"
	"go-micro.dev/v4/api/server/cors"
	"go-micro.dev/v4/logger"
)

type Option func(o *Options)

type Options struct {
	EnableACME   bool
	EnableCORS   bool
	CORSConfig   *cors.Config
	ACMEProvider acme.Provider
	EnableTLS    bool
	ACMEHosts    []string
	TLSConfig    *tls.Config
	Resolver     resolver.Resolver
	Wrappers     []Wrapper
	Logger       logger.Logger
}

type Wrapper func(h http.Handler) http.Handler

func NewOptions(opts ...Option) Options {
	options := Options{
		Logger: logger.DefaultLogger,
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}

func WrapHandler(w Wrapper) Option {
	return func(o *Options) {
		o.Wrappers = append(o.Wrappers, w)
	}
}

func EnableCORS(b bool) Option {
	return func(o *Options) {
		o.EnableCORS = b
	}
}

func CORSConfig(c *cors.Config) Option {
	return func(o *Options) {
		o.CORSConfig = c
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

// Logger sets the underline logging framework.
func Logger(l logger.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}
