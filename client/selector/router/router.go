// Package router is a network/router selector
package router

import (
	"context"
	"net"
	"os"
	"sort"
	"strconv"
	"sync"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/client/selector"
	"github.com/micro/go-micro/network/router"
	pb "github.com/micro/go-micro/network/router/proto"
	"github.com/micro/go-micro/registry"
)

type routerSelector struct {
	opts selector.Options

	// the router
	r router.Router

	// the client for the remote router
	c pb.RouterService

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
		return r.r.Table().Lookup(router.NewQuery(
			router.QueryDestination(service),
		))
	}

	// lookup the remote router

	var clientOpts []client.CallOption

	// set the remote address if specified
	if len(r.addr) > 0 {
		clientOpts = append(clientOpts, client.WithAddress(r.addr))
	}

	// call the router
	pbRoutes, err := r.c.Lookup(context.Background(), &pb.LookupRequest{
		Query: &pb.Query{
			Destination: service,
		},
	}, clientOpts...)
	if err != nil {
		return nil, err
	}

	var routes []router.Route

	// convert from pb to []*router.Route
	for _, r := range pbRoutes.Routes {
		routes = append(routes, router.Route{
			Destination: r.Destination,
			Gateway:     r.Gateway,
			Router:      r.Router,
			Network:     r.Network,
			Metric:      int(r.Metric),
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
		c: pb.NewRouterService(routerName, c),
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
