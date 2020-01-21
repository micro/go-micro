package service

import (
	"context"

	"github.com/micro/go-micro/config/source"
)

type serviceNameKey struct{}
type keyKey struct{}
type pathKey struct{}

func ServiceName(name string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, serviceNameKey{}, name)
	}
}

func Key(key string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, keyKey{}, key)
	}
}

func Path(path string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, pathKey{}, path)
	}
}
