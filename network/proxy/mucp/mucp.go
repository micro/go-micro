// Package mucp transparently forwards the incoming request using a go-micro client.
package mucp

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/client/selector"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/codec/bytes"
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/network/proxy"
	"github.com/micro/go-micro/network/router"
	"github.com/micro/go-micro/server"

	pb "github.com/micro/go-micro/network/router/proto"
)

// Proxy will transparently proxy requests to an endpoint.
// If no endpoint is specified it will call a service using the client.
type Proxy struct {
	// embed options
	options.Options

	// Endpoint specified the fixed service endpoint to call.
	Endpoint string

	// The client to use for outbound requests
	Client client.Client

	// The router for routes
	Router router.Router

	// The router service client
	RouterService pb.RouterService

	// A fib of routes service:address
	sync.RWMutex
	Routes map[string][]router.Route
}

// read client request and write to server
func readLoop(r server.Request, s client.Stream) error {
	// request to backend server
	req := s.Request()

	for {
		// get data from client
		//  no need to decode it
		body, err := r.Read()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		// get the header from client
		hdr := r.Header()
		msg := &codec.Message{
			Type:   codec.Request,
			Header: hdr,
			Body:   body,
		}

		// write the raw request
		err = req.Codec().Write(msg, nil)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}
}

func (p *Proxy) getRoute(service string) ([]string, error) {
	// converts routes to just addresses
	toNodes := func(routes []router.Route) []string {
		var nodes []string
		for _, node := range routes {
			nodes = append(nodes, node.Gateway)
		}
		return nodes
	}

	// lookup the route cache first
	p.RLock()
	routes, ok := p.Routes[service]
	// got it!
	if ok {
		p.RUnlock()
		return toNodes(routes), nil
	}
	p.RUnlock()

	// route cache miss, now lookup the router
	// if it does not exist, don't error out
	// the proxy will just hand off to the client
	// and try the registry
	// in future we might set a default gateway
	if p.Router != nil {
		// lookup the router
		routes, err := p.Router.Table().Lookup(
			router.NewQuery(router.QueryDestination(service)),
		)
		if err != nil {
			return nil, err
		}

		p.Lock()
		if p.Routes == nil {
			p.Routes = make(map[string][]router.Route)
		}
		p.Routes[service] = routes
		p.Unlock()

		return toNodes(routes), nil
	}

	// we've tried getting cached routes
	// we've tried using the router
	addr := os.Getenv("MICRO_ROUTER_ADDRESS")
	name := os.Getenv("MICRO_ROUTER")

	// no router is specified we're going to set the default
	if len(name) == 0 && len(addr) == 0 {
		p.Router = router.DefaultRouter
		go p.Router.Advertise()

		// recursively execute getRoute
		return p.getRoute(service)
	}

	if len(name) == 0 {
		name = "go.micro.router"
	}

	// lookup the remote router

	var addrs []string

	// set the remote address if specified
	if len(addr) > 0 {
		addrs = append(addrs, addr)
	} else {
		// we have a name so we need to check the registry
		services, err := p.Client.Options().Registry.GetService(name)
		if err != nil {
			return nil, err
		}

		for _, service := range services {
			for _, node := range service.Nodes {
				addr := node.Address
				if node.Port > 0 {
					addr = fmt.Sprintf("%s:%d", node.Address, node.Port)
				}
				addrs = append(addrs, addr)
			}
		}
	}

	// no router addresses available
	if len(addrs) == 0 {
		return nil, selector.ErrNoneAvailable
	}

	var pbRoutes *pb.LookupResponse
	var gerr error

	// set default client
	if p.RouterService == nil {
		p.RouterService = pb.NewRouterService(name, p.Client)
	}

	// TODO: implement backoff and retries
	for _, addr := range addrs {
		// call the router
		proutes, err := p.RouterService.Lookup(context.Background(), &pb.LookupRequest{
			Query: &pb.Query{
				Destination: service,
			},
		}, client.WithAddress(addr))
		if err != nil {
			gerr = err
			continue
		}
		// set routes
		pbRoutes = proutes
		break
	}

	// errored out
	if gerr != nil {
		return nil, gerr
	}

	// no routes
	if pbRoutes == nil {
		return nil, selector.ErrNoneAvailable
	}

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

	return toNodes(routes), nil
}

// ServeRequest honours the server.Router interface
func (p *Proxy) ServeRequest(ctx context.Context, req server.Request, rsp server.Response) error {
	// set default client
	if p.Client == nil {
		p.Client = client.DefaultClient
	}

	// service name
	service := req.Service()
	endpoint := req.Endpoint()
	var addresses []string

	// call a specific backend endpoint either by name or address
	if len(p.Endpoint) > 0 {
		// address:port
		if parts := strings.Split(p.Endpoint, ":"); len(parts) > 1 {
			addresses = []string{p.Endpoint}
		} else {
			// get route for endpoint from router
			addr, err := p.getRoute(p.Endpoint)
			if err != nil {
				return err
			}
			// set the address
			addresses = addr
			// set the name
			service = p.Endpoint
		}
	} else {
		// no endpoint was specified just lookup the route
		// get route for endpoint from router
		addr, err := p.getRoute(service)
		if err != nil {
			return err
		}
		addresses = addr
	}

	var opts []client.CallOption

	// set address if available
	if len(addresses) > 0 {
		opts = append(opts, client.WithAddress(addresses...))
	}

	// read initial request
	body, err := req.Read()
	if err != nil {
		return err
	}

	// create new request with raw bytes body
	creq := p.Client.NewRequest(service, endpoint, &bytes.Frame{body}, client.WithContentType(req.ContentType()))

	// create new stream
	stream, err := p.Client.Stream(ctx, creq, opts...)
	if err != nil {
		return err
	}
	defer stream.Close()

	// create client request read loop
	go readLoop(req, stream)

	// get raw response
	resp := stream.Response()

	// create server response write loop
	for {
		// read backend response body
		body, err := resp.Read()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		// read backend response header
		hdr := resp.Header()

		// write raw response header to client
		rsp.WriteHeader(hdr)

		// write raw response body to client
		err = rsp.Write(body)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}

	return nil
}

// NewSingleHostProxy returns a proxy which sends requests to a single backend
func NewSingleHostProxy(endpoint string) *Proxy {
	return &Proxy{
		Options:  options.NewOptions(),
		Endpoint: endpoint,
	}
}

// NewProxy returns a new proxy which will route based on mucp headers
func NewProxy(opts ...options.Option) proxy.Proxy {
	p := new(Proxy)
	p.Options = options.NewOptions(opts...)
	p.Options.Init(options.WithString("mucp"))

	// get endpoint
	ep, ok := p.Options.Values().Get("proxy.endpoint")
	if ok {
		p.Endpoint = ep.(string)
	}

	// get client
	c, ok := p.Options.Values().Get("proxy.client")
	if ok {
		p.Client = c.(client.Client)
	}

	// get router
	r, ok := p.Options.Values().Get("proxy.router")
	if ok {
		p.Router = r.(router.Router)
		// TODO: should we advertise?
		go p.Router.Advertise()
	}

	return p
}
