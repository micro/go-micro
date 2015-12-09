package registry

import (
	"math/rand"
	"time"
)

type randomSelector struct {
	so SelectorOptions
}

func init() {
	rand.Seed(time.Now().Unix())
}

func (r *randomSelector) Select(service string, opts ...SelectOption) (SelectNext, error) {
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

	var nodes []*Node

	for _, service := range services {
		for _, node := range service.Nodes {
			nodes = append(nodes, node)
		}
	}

	if len(nodes) == 0 {
		return nil, ErrNotFound
	}

	return func() (*Node, error) {
		return nodes[rand.Int()%len(nodes)], nil
	}, nil
}

func (r *randomSelector) Mark(service string, node *Node, err error) {
	return
}

func (r *randomSelector) Reset(service string) {
	return
}

func (r *randomSelector) Close() error {
	return nil
}

func NewRandomSelector(opts ...SelectorOption) Selector {
	var sopts SelectorOptions

	for _, opt := range opts {
		opt(&sopts)
	}

	if sopts.Registry == nil {
		sopts.Registry = DefaultRegistry
	}

	return &randomSelector{sopts}
}
