package multi

import (
	"context"

	"github.com/micro/go-micro/v2/registry"
)

type writeKey struct{}
type readKey struct{}

// helper for setting registry options
func setRegistryOption(k, v interface{}) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

// WriteRegistry add underlining registries
func WriteRegistry(w ...registry.Registry) registry.Option {
	return setRegistryOption(writeKey{}, w)
}

// ReadRegistry add underlining registries
func ReadRegistry(r ...registry.Registry) registry.Option {
	return setRegistryOption(readKey{}, r)
}
