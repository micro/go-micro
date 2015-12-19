package roundrobin

import (
	"sync"

	"github.com/micro/go-micro/cmd"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/selector"
)

type roundRobinSelector struct {
	so selector.Options
}

func init() {
	cmd.Selectors["roundrobin"] = NewSelector
}

func (r *roundRobinSelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
	var sopts selector.SelectOptions
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
		return nil, selector.ErrNotFound
	}

	var nodes []*registry.Node

	for _, service := range services {
		for _, node := range service.Nodes {
			nodes = append(nodes, node)
		}
	}

	if len(nodes) == 0 {
		return nil, selector.ErrNotFound
	}

	var i int
	var mtx sync.Mutex

	return func() (*registry.Node, error) {
		mtx.Lock()
		defer mtx.Unlock()
		i++
		return nodes[i%len(nodes)], nil
	}, nil
}

func (r *roundRobinSelector) Mark(service string, node *registry.Node, err error) {
	return
}

func (r *roundRobinSelector) Reset(service string) {
	return
}

func (r *roundRobinSelector) Close() error {
	return nil
}

func (r *roundRobinSelector) String() string {
	return "roundrobin"
}

func NewSelector(opts ...selector.Option) selector.Selector {
	var sopts selector.Options

	for _, opt := range opts {
		opt(&sopts)
	}

	if sopts.Registry == nil {
		sopts.Registry = registry.DefaultRegistry
	}

	return &roundRobinSelector{sopts}
}
