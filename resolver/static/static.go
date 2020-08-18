// Package static is a static resolver
package static

import (
	"github.com/micro/go-micro/v3/resolver"
)

// Resolver returns a static list of nodes. In the event the node list
// is not present it will return the name of the network passed in.
type Resolver struct {
	// A static list of nodes
	Nodes []string
}

// Resolve returns the list of nodes
func (r *Resolver) Resolve(name string) ([]*resolver.Record, error) {
	// if there are no nodes just return the name
	if len(r.Nodes) == 0 {
		return []*resolver.Record{
			{Address: name},
		}, nil
	}

	records := make([]*resolver.Record, 0, len(r.Nodes))

	for _, node := range r.Nodes {
		records = append(records, &resolver.Record{
			Address: node,
		})
	}

	return records, nil
}
