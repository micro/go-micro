// Package router is a network/router selector
package router

import (
	"context"
	"net"
	"sort"
	"strconv"
	"sync"

	"github.com/micro/go-micro/client/selector"
	"github.com/micro/go-micro/network/router"
	"github.com/micro/go-micro/registry"
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
	// lookup router for routes for the service
	routes, err := r.r.Table().Lookup(router.NewQuery(
		router.QueryDestination(service),
	))

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
		address := route.Gateway
		port := 0

		// check if its host:port
		host, pr, err := net.SplitHostPort(address)
		if err == nil {
			pp, _ := strconv.Atoi(pr)
			// set port
			port = pp
			// set address
			address = host
		}

		// return as a node
		return &registry.Node{
			// TODO: add id and metadata if we can
			Address: address,
			Port:    port,
		}, nil
	}, nil
}

func (r *routerSelector) Mark(service string, node *registry.Node, err error) {
	// TODO: pass back metrics or information to the router
	return
}

func (r *routerSelector) Reset(service string) {
	// TODO: reset the metrics or information at the router
	return
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

	// try get from the context
	r, ok := options.Context.Value(routerKey{}).(router.Router)
	if !ok {
		// TODO: Use router.DefaultRouter?
		r = router.NewRouter(
			router.Registry(options.Registry),
		)
	}

	// start the router advertisements
	r.Advertise()

	return &routerSelector{
		opts: options,
		r:    r,
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
