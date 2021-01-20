package mucp

import (
	"context"

	"github.com/asim/go-micro/v3/config/source"
)

type serviceNameKey struct{}
type pathKey struct{}

func WithServiceName(a string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, serviceNameKey{}, a)
	}
}

// WithPath sets the key prefix to use
func WithPath(p string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, pathKey{}, p)
	}
}
