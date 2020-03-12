package service

import (
	"context"

	"github.com/micro/go-micro/v2/config/source"
)

type serviceNameKey struct{}
type namespaceKey struct{}
type pathKey struct{}

func ServiceName(name string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, serviceNameKey{}, name)
	}
}

func Namespace(namespace string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, namespaceKey{}, namespace)
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
