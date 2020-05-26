// Package router is a network/router selector
package router

import (
	"context"
	"os"
	"sort"
	"sync"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/selector"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/router"
	pb "github.com/micro/go-micro/v2/router/service/proto"
)

type routerSelector struct {
	opts selector.Options

	// the router
	r router.Router

	// the client we have
	c client.Client

	// the client for the remote router
	rs pb.RouterService

	// name of the router
	name string

	// address of the remote router
	addr string

	// whether to use the remote router
	remote bool
}

type clientKey struct{}
type routerKey struct{}

// getRoutes returns the routes whether they are remote or local
func (r *routerSelector) getRoutes(service string) ([]router.Route, error) {
	if !r.remote {
		// lookup router for routes for the service
		return r.r.Lookup(
			router.QueryService(service),
		)
	}

	// lookup the remote router

	var addrs []string

	// set the remote address if specified
	if len(r.addr) > 0 {
		addrs = append(addrs, r.addr)
	} else {
		// we have a name so we need to check the registry
		services, err := r.c.Options().Registry.GetService(r.name)
		if err != nil {
			return nil, err
		}

		for _, service := range services {
			for _, node := range service.Nodes {
				addrs = append(addrs, node.Address)
			}
		}
	}

	// no router addresses available
	if len(addrs) == 0 {
		return nil, selector.ErrNoneAvailable
	}

	var pbRoutes *pb.LookupResponse
	var err error

	// TODO: implement backoff and retries
	for _, addr := range addrs {
		// call the router
		pbRoutes, err = r.rs.Lookup(context.Background(), &pb.LookupRequest{
			Query: &pb.Query{
				Service: service,
			},
		}, client.WithAddress(addr))
		if err != nil {
			continue
		}
		break
	}

	// errored out
	if err != nil {
		return nil, err
	}

	// no routes
	if pbRoutes == nil {
		return nil, selector.ErrNoneAvailable
	}

	routes := make([]router.Route, 0, len(pbRoutes.Routes))

	// convert from pb to []*router.Route
	for _, r := range pbRoutes.Routes {
		routes = append(routes, router.Route{
			Service: r.Service,
			Address: r.Address,
			Gateway: r.Gateway,
			Network: r.Network,
			Link:    r.Link,
			Metric:  r.Metric,
		})
	}

	return routes, nil
}

func (r *routerSelector) Init(opts ...selector.Option) error {
	// no op
	return nil
}

func (r *routerSelector) Options() selector.Options {
	return r.opts
}

func (r *routerSelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
	// TODO: pull routes asynchronously and cache
	routes, err := r.getRoutes(service)
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

	// try get client from the context
	c, ok := options.Context.Value(clientKey{}).(client.Client)
	if !ok {
		c = client.DefaultClient
	}

	// get the router from env vars if its a remote service
	remote := true
	routerName := os.Getenv("MICRO_ROUTER")
	routerAddress := os.Getenv("MICRO_ROUTER_ADDRESS")

	// start the router advertisements if we're running it locally
	if len(routerName) == 0 && len(routerAddress) == 0 {
		go r.Advertise()
		remote = false
	}

	return &routerSelector{
		opts: options,
		// set the internal router
		r: r,
		// set the client
		c: c,
		// set the router client
		rs: pb.NewRouterService(routerName, c),
		// name of the router
		name: routerName,
		// address of router
		addr: routerAddress,
		// let ourselves know to use the remote router
		remote: remote,
	}
}

// WithClient sets the client for the request
func WithClient(c client.Client) selector.Option {
	return func(o *selector.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, clientKey{}, c)
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
