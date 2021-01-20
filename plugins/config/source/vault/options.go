package vault

import (
	"context"

	"github.com/asim/go-micro/v3/config/source"
)

type addressKey struct{}
type resourcePath struct{}
type nameSpace struct{}
type tokenKey struct{}
type secretName struct{}

// WithAddress sets the server address
func WithAddress(a string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, addressKey{}, a)
	}
}

// WithResourcePath sets the resource that will be access
func WithResourcePath(p string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, resourcePath{}, p)
	}
}

// WithNameSpace sets the namespace that its going to be access
func WithNameSpace(n string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, nameSpace{}, n)
	}
}

// WithToken sets the key token to use
func WithToken(t string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, tokenKey{}, t)
	}
}

// WithSecretName sets the name of the secret to wrap in on a map
func WithSecretName(t string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, secretName{}, t)
	}
}
