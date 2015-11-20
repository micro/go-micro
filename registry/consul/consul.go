package consul

// This is a hack

import (
	"github.com/micro/go-micro/registry"
)

func NewRegistry(addrs []string, opt ...registry.Option) registry.Registry {
	return registry.NewRegistry(addrs, opt...)
}
