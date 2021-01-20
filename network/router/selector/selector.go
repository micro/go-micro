package selector

import (
	"context"
	"sort"
	"sync"

	"github.com/asim/go-micro/v3/network/router"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/selector"
)

type routerSelector struct {
	opts selector.Options

	// the router
	r router.Router
}

type routerKey struct{}

func (r *routerSelector) Init(opts ...selector.Option) error {
	// no op
	return nil
}

func (r *routerSelector) Options() selector.Options {
	return r.opts
}

func (r *routerSelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
	// TODO: pull routes asynchronously and cache
	routes, err := r.r.Lookup(
		router.QueryService(service),
	)
	if err != nil {
		return nil, err
	}

	// no routes return not found error
	if len(routes) == 0 {
		return nil, selector.ErrNotFound
	}

	// TODO: apply filters by pseudo constructing service

	// sort the routes based on metric
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Metric < routes[j].Metric
	})

	// roundrobin assuming routes are in metric preference order
	var i int
	var mtx sync.Mutex

	return func() (*registry.Node, error) {
		// get index and increment counter with every call to next
		mtx.Lock()
		idx := i
		i++
		mtx.Unlock()

		// get route based on idx
		route := routes[idx%len(routes)]

		// defaults to gateway and no port
		address := route.Address
		if len(route.Gateway) > 0 {
			address = route.Gateway
		}

		// return as a node
		return &registry.Node{
			// TODO: add id and metadata if we can
			Address: address,
		}, nil
	}, nil
}

func (r *routerSelector) Mark(service string, node *registry.Node, err error) {
	// TODO: pass back metrics or information to the router
}

func (r *routerSelector) Reset(service string) {
	// TODO: reset the metrics or information at the router
}

func (r *routerSelector) Close() error {
	// stop the router advertisements
	return r.r.Stop()
}

func (r *routerSelector) String() string {
	return "router"
}

// NewSelector returns a new router based selector
func NewSelector(opts ...selector.Option) selector.Selector {
	options := selector.Options{
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}

	// set default registry if not set
	if options.Registry == nil {
		options.Registry = registry.DefaultRegistry
	}

	// try get router from the context
	r, ok := options.Context.Value(routerKey{}).(router.Router)
	if !ok {
		// TODO: Use router.DefaultRouter?
		r = router.NewRouter(
			router.Registry(options.Registry),
		)
	}

	go r.Advertise()

	return &routerSelector{
		opts: options,
		// set the internal router
		r: r,
	}
}

// WithRouter sets the router as an option
func WithRouter(r router.Router) selector.Option {
	return func(o *selector.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, routerKey{}, r)
	}
}
