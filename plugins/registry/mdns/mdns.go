// Package mdns provides a multicast dns registry
package mdns

import (
	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/registry"
)

func init() {
	cmd.DefaultRegistries["mdns"] = NewRegistry
}

// NewRegistry returns a new mdns registry
func NewRegistry(opts ...registry.Option) registry.Registry {
	return registry.NewRegistry(opts...)
}

