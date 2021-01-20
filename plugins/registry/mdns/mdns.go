// Package mdns provides a multicast dns registry
package mdns

import (
	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/registry/mdns"
)

func init() {
	cmd.DefaultRegistries["mdns"] = NewRegistry
}

// NewRegistry returns a new mdns registry
func NewRegistry(opts ...registry.Option) registry.Registry {
	return registry.NewRegistry(opts...)
}

// Domain sets the mdnsDomain
func Domain(d string) registry.Option {
	return mdns.Domain(d)
}
