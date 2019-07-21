// Package mucp transparently forwards the incoming request using a go-micro client.
package mucp

import (
	"context"
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
	"github.com/micro/go-micro/network/router/table"
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
	Routes map[string]map[uint64]table.Route

	// The channel to monitor watcher errors
	errChan chan error
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
	toNodes := func(routes map[uint64]table.Route) []string {
		var nodes []string
		for _, node := range routes {
			address := node.Address
			if len(node.Gateway) > 0 {
				address = node.Gateway
			}
			nodes = append(nodes, address)
		}
		return nodes
	}

	// lookup the route cache first
	p.RLock()
	if p.Routes == nil {
		p.Routes = make(map[string]map[uint64]table.Route)
	}
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
		result, err := p.Router.Lookup(
			table.NewQuery(table.QueryService(service)),
		)
		if err != nil {
			return nil, err
		}

		// update the proxy cache
		p.Lock()
		for _, route := range result {
			p.Routes[service][route.Hash()] = route
		}
		routes = p.Routes[service]
		p.Unlock()

		return toNodes(routes), nil
	}

	// we've tried getting cached routes
	// we've tried using the router
	addr := os.Getenv("MICRO_ROUTER_ADDRESS")
	name := os.Getenv("MICRO_ROUTER")

	// no router is specified we're going to set the default
	if len(name) == 0 && len(addr) == 0 {
		// create default router and start it
		p.Router = router.DefaultRouter
		if err := p.Router.Run(); err != nil {
			return nil, err
		}

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
				addrs = append(addrs, node.Address)
			}
		}
	}

	// no router addresses available
	if len(addrs) == 0 {
		return nil, selector.ErrNoneAvailable
	}

	// set default client
	if p.RouterService == nil {
		p.RouterService = pb.NewRouterService(name, p.Client)
	}

	var resp *pb.LookupResponse
	var err error

	// TODO: implement backoff and retries
	for _, addr := range addrs {
		// call the router
		resp, err = p.RouterService.Lookup(context.Background(), &pb.LookupRequest{
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
	if resp == nil || len(resp.Routes) == 0 {
		return nil, selector.ErrNoneAvailable
	}

	// convert from pb to []*router.Route
	for _, r := range resp.Routes {
		route := table.Route{
			Service: r.Service,
			Address: r.Address,
			Gateway: r.Gateway,
			Network: r.Network,
			Link:    r.Link,
			Metric:  int(r.Metric),
		}
		routes[route.Hash()] = route
	}

	return toNodes(routes), nil
}

// watchRoutes watches remote router service routes and updates proxy cache
func (p *Proxy) watchRoutes() {
	// this is safe to do as the only way watchRoutes returns is
	// when some error is written into error channel - we want to bail then
	defer close(p.errChan)

	stream, err := p.RouterService.Watch(context.Background(), &pb.WatchRequest{})
	if err != nil {
		p.errChan <- err
		return
	}
	defer stream.Close()

	for {
		event, err := stream.Recv()
		if err != nil {
			p.errChan <- err
			return
		}

		p.Lock()
		if p.Routes == nil {
			p.Routes = make(map[string]map[uint64]table.Route)
		}
		route := table.Route{
			Service: event.Route.Service,
			Address: event.Route.Address,
			Gateway: event.Route.Gateway,
			Network: event.Route.Network,
			Link:    event.Route.Link,
			Metric:  int(event.Route.Metric),
		}
		p.Routes[route.Service][route.Hash()] = route
		p.Unlock()
	}
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

	// route watcher error
	var wrErr error

	// create server response write loop
	for {
		select {
		case err := <-p.errChan:
			if err != nil {
				wrErr = err
			}
			return wrErr
		default:
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
	}

	// watch router service routes
	p.errChan = make(chan error, 1)
	go p.watchRoutes()

	return p
}
