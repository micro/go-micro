// Package consul provides a consul based registry and is the default discovery system
package consul

import (
	"github.com/micro/go-micro/registry"
)

// NewRegistry returns a new consul registry
func NewRegistry(opts ...registry.Option) registry.Registry {
	return registry.NewRegistry(opts...)
}
