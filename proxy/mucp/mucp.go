// Package mucp transparently forwards the incoming request using a go-micro client.
package mucp

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/codec/bytes"
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/proxy"
	"github.com/micro/go-micro/router"
	"github.com/micro/go-micro/server"
)

// Proxy will transparently proxy requests to an endpoint.
// If no endpoint is specified it will call a service using the client.
type Proxy struct {
	// embed options
	options.Options

	// Endpoint specifies the fixed service endpoint to call.
	Endpoint string

	// The client to use for outbound requests
	Client client.Client

	// The router for routes
	Router router.Router

	// A fib of routes service:address
	sync.RWMutex
	Routes map[string]map[uint64]router.Route

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

// toNodes returns a list of node addresses from given routes
func toNodes(routes map[uint64]router.Route) []string {
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

func (p *Proxy) getRoute(service string) ([]string, error) {
	// lookup the route cache first
	p.Lock()
	routes, ok := p.Routes[service]
	if ok {
		p.Unlock()
		return toNodes(routes), nil
	}
	p.Routes[service] = make(map[uint64]router.Route)
	p.Unlock()

	// if the router is broken return error
	if status := p.Router.Status(); status.Code == router.Error {
		return nil, status.Error
	}

	// lookup the routes in the router
	results, err := p.Router.Lookup(router.NewQuery(router.QueryService(service)))
	if err != nil {
		return nil, err
	}

	// update the proxy cache
	p.Lock()
	for _, route := range results {
		p.Routes[service][route.Hash()] = route
	}
	routes = p.Routes[service]
	p.Unlock()

	return toNodes(routes), nil
}

// manageRouteCache applies action on a given route to Proxy route cache
func (p *Proxy) manageRouteCache(route router.Route, action string) error {
	switch action {
	case "create", "update":
		if _, ok := p.Routes[route.Service]; !ok {
			p.Routes[route.Service] = make(map[uint64]router.Route)
		}
		p.Routes[route.Service][route.Hash()] = route
	case "delete":
		if _, ok := p.Routes[route.Service]; !ok {
			return fmt.Errorf("route not found")
		}
		delete(p.Routes[route.Service], route.Hash())
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	return nil
}

// watchRoutes watches service routes and updates proxy cache
func (p *Proxy) watchRoutes() {
	// this is safe to do as the only way watchRoutes returns is
	// when some error is written into error channel - we want to bail then
	defer close(p.errChan)

	// route watcher
	w, err := p.Router.Watch()
	if err != nil {
		p.errChan <- err
		return
	}

	for {
		event, err := w.Next()
		if err != nil {
			p.errChan <- err
			return
		}

		p.Lock()
		if err := p.manageRouteCache(event.Route, fmt.Sprintf("%s", event.Type)); err != nil {
			// TODO: should we bail here?
			p.Unlock()
			continue
		}
		p.Unlock()
	}
}

// ServeRequest honours the server.Router interface
func (p *Proxy) ServeRequest(ctx context.Context, req server.Request, rsp server.Response) error {
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
	var watchErr error

	// create server response write loop
	for {
		select {
		case err := <-p.errChan:
			if err != nil {
				watchErr = err
			}
			return watchErr
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

	// set the default client
	if p.Client == nil {
		p.Client = client.DefaultClient
	}

	// get router
	r, ok := p.Options.Values().Get("proxy.router")
	if ok {
		p.Router = r.(router.Router)
	}

	// create default router and start it
	if p.Router == nil {
		p.Router = router.DefaultRouter
	}

	// routes cache
	p.Routes = make(map[string]map[uint64]router.Route)

	// watch router service routes
	p.errChan = make(chan error, 1)
	go p.watchRoutes()

	return p
}
