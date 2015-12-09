package registry

import (
	"sync"
)

type roundRobinSelector struct {
	so SelectorOptions
}

func (r *roundRobinSelector) Select(service string, opts ...SelectOption) (SelectNext, error) {
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

	var i int
	var mtx sync.Mutex

	return func() (*Node, error) {
		mtx.Lock()
		defer mtx.Unlock()
		i++
		return nodes[i%len(nodes)], nil
	}, nil
}

func (r *roundRobinSelector) Mark(service string, node *Node, err error) {
	return
}

func (r *roundRobinSelector) Reset(service string) {
	return
}

func (r *roundRobinSelector) Close() error {
	return nil
}

func NewRoundRobinSelector(opts ...SelectorOption) Selector {
	var sopts SelectorOptions

	for _, opt := range opts {
		opt(&sopts)
	}

	if sopts.Registry == nil {
		sopts.Registry = DefaultRegistry
	}

	return &roundRobinSelector{sopts}
}
