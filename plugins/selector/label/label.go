// Package label is a priority label based selector.
package label

import (
	"context"
	"sync"

	"github.com/asim/go-micro/v3/selector"
	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/registry"
)

/*
   A priority based label selector. Rather than just returning nodes with specific labels
   this selector orders the nodes based on a list of labels. If no labels match all the
   nodes are still returned. The priority based label selector is useful for such things
   as rudimentary AZ based routing where requests made to other services should remain
   in the same AZ.
*/

type labelSelector struct {
	so selector.Options
}

func init() {
	cmd.DefaultSelectors["label"] = NewSelector
}

func prioritise(nodes []*registry.Node, labels []label) []*registry.Node {
	var lnodes []*registry.Node
	marked := make(map[string]bool)

	for _, label := range labels {
		for _, node := range nodes {
			// already used
			if _, ok := marked[node.Id]; ok {
				continue
			}

			// nil metadata?
			if node.Metadata == nil {
				continue
			}

			// matching label?
			if val, ok := node.Metadata[label.key]; !ok || label.val != val {
				continue
			}

			// matched! mark it
			marked[node.Id] = true

			// append to nodes
			lnodes = append(lnodes, node)
		}
	}

	// grab the leftovers
	for _, node := range nodes {
		if _, ok := marked[node.Id]; ok {
			continue
		}
		lnodes = append(lnodes, node)
	}

	return lnodes
}

func next(nodes []*registry.Node) func() (*registry.Node, error) {
	var i int
	var mtx sync.Mutex

	return func() (*registry.Node, error) {
		mtx.Lock()
		if i >= len(nodes) {
			i = 0
		}
		node := nodes[i]
		i++
		mtx.Unlock()
		return node, nil
	}
}

func (r *labelSelector) Init(opts ...selector.Option) error {
	for _, o := range opts {
		o(&r.so)
	}
	return nil
}

func (r *labelSelector) Options() selector.Options {
	return r.so
}

func (r *labelSelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
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

	// flatten node list
	for _, service := range services {
		for _, node := range service.Nodes {
			nodes = append(nodes, node)
		}
	}

	// any nodes left?
	if len(nodes) == 0 {
		return nil, selector.ErrNotFound
	}

	// now prioritise the list based on labels
	// oh god the O(n)^2 cruft or well not really
	// more like O(m*n) or something like that
	if labels, ok := r.so.Context.Value(labelKey{}).([]label); ok {
		nodes = prioritise(nodes, labels)
	}

	return next(nodes), nil
}

func (r *labelSelector) Mark(service string, node *registry.Node, err error) {
	return
}

func (r *labelSelector) Reset(service string) {
	return
}

func (r *labelSelector) Close() error {
	return nil
}

func (r *labelSelector) String() string {
	return "label"
}

func NewSelector(opts ...selector.Option) selector.Selector {
	sopts := selector.Options{
		Context:  context.TODO(),
		Registry: registry.DefaultRegistry,
	}

	for _, opt := range opts {
		opt(&sopts)
	}

	return &labelSelector{sopts}
}
