package memory

import (
	"context"

	"github.com/micro/go-micro/registry"
)

type servicesKey struct{}

func getServices(ctx context.Context) map[string][]*registry.Service {
	s, ok := ctx.Value(servicesKey{}).(map[string][]*registry.Service)
	if !ok {
		return nil
	}
	return s
}

// Services is an option that preloads service data
func Services(s map[string][]*registry.Service) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, servicesKey{}, s)
	}
}
