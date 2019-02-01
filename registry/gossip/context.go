package gossip

import (
	"context"

	"github.com/micro/go-micro/registry"
)

// setRegistryOption returns a function to setup a context with given value
func setRegistryOption(k, v interface{}) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}
