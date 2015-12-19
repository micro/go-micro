/*
The Selector package provides a way to algorithmically filter and return
nodes required by the client or any other system. Selector's implemented
by Micro build on the registry but it's of optional use. One could
provide a static Selector that has a fixed pool.

	func (r *randomSelector) Select(service string, opts ...SelectOption) (Next, error) {
		var sopts SelectOptions
		for _, opt := range opts {
			opt(&sopts)
		}

		// get the service
		services, err := r.so.Registry.GetService(service)
		if err != nil {
			return nil, err
		}

		// apply the filters
		for _, filter := range sopts.Filters {
			services = filter(services)
		}

		// if there's nothing left, return
		if len(services) == 0 {
			return nil, ErrNotFound
		}

		var nodes []*registry.Node

		for _, service := range services {
			for _, node := range service.Nodes {
				nodes = append(nodes, node)
			}
		}

		if len(nodes) == 0 {
			return nil, ErrNotFound
		}

		return func() (*registry.Node, error) {
			i := rand.Int()
			j := i % len(services)

			if len(services[j].Nodes) == 0 {
				return nil, ErrNotFound
			}

			k := i % len(services[j].Nodes)
			return services[j].Nodes[k], nil
		}, nil
	}


*/
package selector

import (
	"errors"
	"github.com/micro/go-micro/registry"
)

// Selector builds on the registry as a mechanism to pick nodes
// and mark their status. This allows host pools and other things
// to be built using various algorithms.
type Selector interface {
	// Select returns a function which should return the next node
	Select(service string, opts ...SelectOption) (Next, error)
	// Mark sets the success/error against a node
	Mark(service string, node *registry.Node, err error)
	// Reset returns state back to zero for a service
	Reset(service string)
	// Close renders the selector unusable
	Close() error
	// Name of the selector
	String() string
}

// Next is a function that returns the next node
// based on the selector's algorithm
type Next func() (*registry.Node, error)

// SelectFilter is used to filter a service during the selection process
type SelectFilter func([]*registry.Service) []*registry.Service

var (
	DefaultSelector = newRandomSelector()

	ErrNotFound      = errors.New("not found")
	ErrNoneAvailable = errors.New("none available")
)

func NewSelector(opts ...Option) Selector {
	return newRandomSelector(opts...)
}
