package consul

import (
	"context"

	"github.com/hashicorp/consul/api"
	"github.com/micro/go-micro/v2/config/source"
)

type addressKey struct{}
type prefixKey struct{}
type stripPrefixKey struct{}
type dcKey struct{}
type tokenKey struct{}
type configKey struct{}

// WithAddress sets the consul address
func WithAddress(a string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, addressKey{}, a)
	}
}

// WithPrefix sets the key prefix to use
func WithPrefix(p string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, prefixKey{}, p)
	}
}

// StripPrefix indicates whether to remove the prefix from config entries, or leave it in place.
func StripPrefix(strip bool) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}

		o.Context = context.WithValue(o.Context, stripPrefixKey{}, strip)
	}
}

func WithDatacenter(p string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, dcKey{}, p)
	}
}

// WithToken sets the key token to use
func WithToken(p string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, tokenKey{}, p)
	}
}

// WithConfig set consul-specific options
func WithConfig(c *api.Config) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, configKey{}, c)
	}
}
