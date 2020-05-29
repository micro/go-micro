package registry

import (
	"context"
	"crypto/tls"
	"time"
)

type Options struct {
	Addrs     []string
	Timeout   time.Duration
	Secure    bool
	TLSConfig *tls.Config
	// Domain to default to
	Domain string
	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type RegisterOptions struct {
	TTL time.Duration
	// Domain the service is running in
	Domain string
	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type WatchOptions struct {
	// Domain to watch
	Domain string
	// Specify a service to watch
	// If blank, the watch is for all services
	Service string
	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type DeregisterOptions struct {
	// Domain the service is running in
	Domain  string
	Context context.Context
}

type GetOptions struct {
	// Domain the service is running in
	Domain  string
	Context context.Context
}

type ListOptions struct {
	// Domain to list from
	Domain  string
	Context context.Context
}

// Addrs is the registry addresses to use
func Addrs(addrs ...string) Option {
	return func(o *Options) {
		o.Addrs = addrs
	}
}

func Timeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}

// Secure communication with the registry
func Secure(b bool) Option {
	return func(o *Options) {
		o.Secure = b
	}
}

// Specify TLS Config
func TLSConfig(t *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = t
	}
}

// Domain to default to
func Domain(d string) Option {
	return func(o *Options) {
		o.Domain = d
	}
}

func RegisterTTL(t time.Duration) RegisterOption {
	return func(o *RegisterOptions) {
		o.TTL = t
	}
}

func RegisterContext(ctx context.Context) RegisterOption {
	return func(o *RegisterOptions) {
		o.Context = ctx
	}
}

func RegisterDomain(d string) RegisterOption {
	return func(o *RegisterOptions) {
		o.Domain = d
	}
}

// Watch a service
func WatchService(name string) WatchOption {
	return func(o *WatchOptions) {
		o.Service = name
	}
}

func WatchContext(ctx context.Context) WatchOption {
	return func(o *WatchOptions) {
		o.Context = ctx
	}
}

func WatchDomain(d string) WatchOption {
	return func(o *WatchOptions) {
		o.Domain = d
	}
}

func DeregisterContext(ctx context.Context) DeregisterOption {
	return func(o *DeregisterOptions) {
		o.Context = ctx
	}
}

func DeregisterDomain(d string) DeregisterOption {
	return func(o *DeregisterOptions) {
		o.Domain = d
	}
}

func GetContext(ctx context.Context) GetOption {
	return func(o *GetOptions) {
		o.Context = ctx
	}
}

func GetDomain(d string) GetOption {
	return func(o *GetOptions) {
		o.Domain = d
	}
}

func ListContext(ctx context.Context) ListOption {
	return func(o *ListOptions) {
		o.Context = ctx
	}
}

func ListDomain(d string) ListOption {
	return func(o *ListOptions) {
		o.Domain = d
	}
}
