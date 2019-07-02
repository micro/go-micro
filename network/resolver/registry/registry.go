// Package registry resolves ids using the go-micro registry
package registry

import (
	"fmt"

	"github.com/micro/go-micro/network/resolver"
	"github.com/micro/go-micro/registry"
)

type Resolver struct {
	// Registry is the registry to use otherwise we use the defaul
	Registry registry.Registry
}

// Resolve assumes ID is a domain name e.g micro.mu
func (r *Resolver) Resolve(id string) ([]*resolver.Record, error) {
	reg := r.Registry
	if reg == nil {
		reg = registry.DefaultRegistry
	}

	services, err := reg.GetService(id)
	if err != nil {
		return nil, err
	}

	var records []*resolver.Record

	for _, service := range services {
		for _, node := range service.Nodes {
			addr := node.Address
			// such a hack
			if node.Port > 0 {
				addr = fmt.Sprintf("%s:%d", node.Address, node.Port)
			}
			records = append(records, &resolver.Record{
				Address: addr,
			})
		}
	}

	return records, nil
}
