// Package registry resolves names using the go-micro registry
package registry

import (
	"github.com/asim/go-micro/v3/network/resolver"
	"github.com/asim/go-micro/v3/registry"
)

// Resolver is a registry network resolver
type Resolver struct {
	// Registry is the registry to use otherwise we use the defaul
	Registry registry.Registry
}

// Resolve assumes ID is a domain name e.g micro.mu
func (r *Resolver) Resolve(name string) ([]*resolver.Record, error) {
	reg := r.Registry
	if reg == nil {
		reg = registry.DefaultRegistry
	}

	services, err := reg.GetService(name)
	if err != nil {
		return nil, err
	}

	var records []*resolver.Record

	for _, service := range services {
		for _, node := range service.Nodes {
			records = append(records, &resolver.Record{
				Address: node.Address,
			})
		}
	}

	return records, nil
}
