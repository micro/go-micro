package memory

import (
	"context"

	"github.com/micro/go-micro/v2/registry"
)

type servicesKey struct{}

func getServiceRecords(ctx context.Context) map[string]map[string]*record {
	memServices, ok := ctx.Value(servicesKey{}).(map[string][]*registry.Service)
	if !ok {
		return nil
	}

	services := make(map[string]map[string]*record)

	for name, svc := range memServices {
		if _, ok := services[name]; !ok {
			services[name] = make(map[string]*record)
		}
		// go through every version of the service
		for _, s := range svc {
			services[s.Name][s.Version] = serviceToRecord(s, 0)
		}
	}

	return services
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
