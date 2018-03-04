package consul

import (
	"context"

	consul "github.com/hashicorp/consul/api"
	"github.com/micro/go-micro/registry"
)

func Config(c *consul.Config) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, "consul_config", c)
	}
}
