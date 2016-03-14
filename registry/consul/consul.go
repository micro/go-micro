package consul

import (
	"github.com/micro/go-micro/cmd"
	"github.com/micro/go-micro/registry"
)

func init() {
	cmd.DefaultRegistries["consul"] = NewRegistry
}

func NewRegistry(addrs []string, opts ...registry.Option) registry.Registry {
	return registry.NewRegistry(addrs, opts...)
}
