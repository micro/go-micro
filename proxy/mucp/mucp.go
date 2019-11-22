// Package mucp transparently forwards the incoming request using a go-micro client.
package mucp

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/client/selector"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/codec/bytes"
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/proxy"
	"github.com/micro/go-micro/router"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-micro/util/log"
)

// Proxy will transparently proxy requests to an endpoint.
// If no endpoint is specified it will call a service using the client.
type Proxy struct {
	// embed options
	options.Options

	// Endpoint specifies the fixed service endpoint to call.
	Endpoint string

	// The client to use for outbound requests in the local network
	Client client.Client

	// Links are used for outbound requests not in the local network
	Links map[string]client.Client

	// The router for routes
	Router router.Router

	// A fib of routes service:address
	sync.RWMutex
	Routes map[string]map[uint64]router.Route
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
func toNodes(routes []router.Route) []string {
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

func toSlice(r map[uint64]router.Route) []router.Route {
	routes := make([]router.Route, 0, len(r))
	for _, v := range r {
		routes = append(routes, v)
	}

	// sort the routes in order of metric
	sort.Slice(routes, func(i, j int) bool { return routes[i].Metric < routes[j].Metric })

	return routes
}

func (p *Proxy) filterRoutes(ctx context.Context, routes []router.Route) []router.Route {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		return routes
	}

	var filteredRoutes []router.Route

	// filter the routes based on our headers
	for _, route := range routes {
		// process only routes for this id
		if id := md["Micro-Router"]; len(id) > 0 {
			if route.Router != id {
				// skip routes that don't mwatch
				continue
			}
		}

		// only process routes with this network
		if net := md["Micro-Network"]; len(net) > 0 {
			if route.Network != net {
				// skip routes that don't mwatch
				continue
			}
		}

		// process only this gateway
		if gw := md["Micro-Gateway"]; len(gw) > 0 {
			// if the gateway matches our address
			// special case, take the routes with no gateway
			// TODO: should we strip the gateway from the context?
			if gw == p.Router.Options().Address {
				if len(route.Gateway) > 0 && route.Gateway != gw {
					continue
				}
				// otherwise its a local route and we're keeping it
			} else {
				// gateway does not match our own
				if route.Gateway != gw {
					continue
				}
			}
		}

		// TODO: address based filtering
		// address := md["Micro-Address"]

		// TODO: label based filtering
		// requires new field in routing table : route.Labels

		// passed the filter checks
		filteredRoutes = append(filteredRoutes, route)
	}

	return filteredRoutes
}

func (p *Proxy) getLink(r router.Route) (client.Client, error) {
	if r.Link == "local" || len(p.Links) == 0 {
		return p.Client, nil
	}
	l, ok := p.Links[r.Link]
	if !ok {
		return nil, errors.InternalServerError("go.micro.proxy", "link not found")
	}
	return l, nil
}

func (p *Proxy) getRoute(ctx context.Context, service string) ([]router.Route, error) {
	// lookup the route cache first
	p.Lock()
	cached, ok := p.Routes[service]
	if ok {
		p.Unlock()
		routes := toSlice(cached)
		return p.filterRoutes(ctx, routes), nil
	}
	p.Unlock()

	// cache routes for the service
	routes, err := p.cacheRoutes(service)
	if err != nil {
		return nil, err
	}

	return p.filterRoutes(ctx, routes), nil
}

func (p *Proxy) cacheRoutes(service string) ([]router.Route, error) {
	// lookup the routes in the router
	results, err := p.Router.Lookup(router.QueryService(service))
	if err != nil {
		// check the status of the router
		if status := p.Router.Status(); status.Code == router.Error {
			return nil, status.Error
		}
		// otherwise return the error
		return nil, err
	}

	// update the proxy cache
	p.Lock()
	for _, route := range results {
		// create if does not exist
		if _, ok := p.Routes[service]; !ok {
			p.Routes[service] = make(map[uint64]router.Route)
		}
		p.Routes[service][route.Hash()] = route
	}
	routes := p.Routes[service]
	p.Unlock()

	return toSlice(routes), nil
}

// refreshMetrics will refresh any metrics for our local cached routes.
// we may not receive new watch events for these as they change.
func (p *Proxy) refreshMetrics() {
	services := make([]string, 0, len(p.Routes))

	// get a list of services to update
	p.RLock()
	for service := range p.Routes {
		services = append(services, service)
	}
	p.RUnlock()

	// get and cache the routes for the service
	for _, service := range services {
		p.cacheRoutes(service)
	}
}

// manageRoutes applies action on a given route to Proxy route cache
func (p *Proxy) manageRoutes(route router.Route, action string) error {
	// we only cache what we are actually concerned with
	p.Lock()
	defer p.Unlock()

	switch action {
	case "create", "update":
		if _, ok := p.Routes[route.Service]; !ok {
			return fmt.Errorf("not called %s", route.Service)
		}
		p.Routes[route.Service][route.Hash()] = route
	case "delete":
		delete(p.Routes[route.Service], route.Hash())
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	return nil
}

// watchRoutes watches service routes and updates proxy cache
func (p *Proxy) watchRoutes() {
	// route watcher
	w, err := p.Router.Watch()
	if err != nil {
		return
	}

	for {
		event, err := w.Next()
		if err != nil {
			return
		}

		if err := p.manageRoutes(event.Route, fmt.Sprintf("%s", event.Type)); err != nil {
			// TODO: should we bail here?
			continue
		}
	}
}

// ProcessMessage acts as a message exchange and forwards messages to ongoing topics
// TODO: should we look at p.Endpoint and only send to the local endpoint? probably
func (p *Proxy) ProcessMessage(ctx context.Context, msg server.Message) error {
	// TODO: check that we're not broadcast storming by sending to the same topic
	// that we're actually subscribed to

	log.Tracef("Received message for %s", msg.Topic())

	var errors []string

	// directly publish to the local client
	if err := p.Client.Publish(ctx, msg); err != nil {
		errors = append(errors, err.Error())
	}

	// publish to all links
	for _, client := range p.Links {
		if err := client.Publish(ctx, msg); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) == 0 {
		return nil
	}

	// there is no error...muahaha
	return fmt.Errorf("Message processing error: %s", strings.Join(errors, "\n"))
}

// ServeRequest honours the server.Router interface
func (p *Proxy) ServeRequest(ctx context.Context, req server.Request, rsp server.Response) error {
	// determine if its local routing
	var local bool
	// address to call
	var addresses []string
	// routes
	var routes []router.Route
	// service name to call
	service := req.Service()
	// endpoint to call
	endpoint := req.Endpoint()

	if len(service) == 0 {
		return errors.BadRequest("go.micro.proxy", "service name is blank")
	}

	log.Tracef("Received request for %s", service)

	// are we network routing or local routing
	if len(p.Links) == 0 {
		local = true
	}

	// call a specific backend endpoint either by name or address
	if len(p.Endpoint) > 0 {
		// address:port
		if parts := strings.Split(p.Endpoint, ":"); len(parts) > 1 {
			addresses = []string{p.Endpoint}
		} else {
			// get route for endpoint from router
			addr, err := p.getRoute(ctx, p.Endpoint)
			if err != nil {
				return err
			}
			// set the address
			routes = addr
			// set the name
			service = p.Endpoint
		}
	} else {
		// no endpoint was specified just lookup the route
		// get route for endpoint from router
		addr, err := p.getRoute(ctx, service)
		if err != nil {
			return err
		}
		routes = addr
	}

	var opts []client.CallOption

	// set strategy to round robin
	opts = append(opts, client.WithSelectOption(selector.WithStrategy(selector.RoundRobin)))

	// if the address is already set just serve it
	// TODO: figure it out if we should know to pick a link
	if len(addresses) > 0 {
		opts = append(opts, client.WithAddress(addresses...))

		// serve the normal way
		return p.serveRequest(ctx, p.Client, service, endpoint, req, rsp, opts...)
	}

	// there's no links e.g we're local routing then just serve it with addresses
	if local {
		var opts []client.CallOption

		// set address if available via routes or specific endpoint
		if len(routes) > 0 {
			addresses := toNodes(routes)
			opts = append(opts, client.WithAddress(addresses...))
		}

		// serve the normal way
		return p.serveRequest(ctx, p.Client, service, endpoint, req, rsp, opts...)
	}

	var gerr error

	// we're routing globally with multiple links
	// so we need to pick a link per route
	for _, route := range routes {
		// pick the link or error out
		link, err := p.getLink(route)
		if err != nil {
			// ok let's try again
			gerr = err
			continue
		}

		log.Debugf("Proxy using route %+v\n", route)

		// set the address to call
		addresses := toNodes([]router.Route{route})
		opts = append(opts, client.WithAddress(addresses...))

		// do the request with the link
		gerr = p.serveRequest(ctx, link, service, endpoint, req, rsp, opts...)
		// return on no error since we succeeded
		if gerr == nil {
			return nil
		}

		// return where the context deadline was exceeded
		if gerr == context.Canceled || gerr == context.DeadlineExceeded {
			return err
		}

		// otherwise attempt to do it all over again
	}

	// if we got here something went really badly wrong
	return gerr
}

func (p *Proxy) serveRequest(ctx context.Context, link client.Client, service, endpoint string, req server.Request, rsp server.Response, opts ...client.CallOption) error {
	// read initial request
	body, err := req.Read()
	if err != nil {
		return err
	}

	// create new request with raw bytes body
	creq := link.NewRequest(service, endpoint, &bytes.Frame{body}, client.WithContentType(req.ContentType()))

	// not a stream so make a client.Call request
	if !req.Stream() {
		crsp := new(bytes.Frame)

		// make a call to the backend
		if err := link.Call(ctx, creq, crsp, opts...); err != nil {
			return err
		}

		// write the response
		if err := rsp.Write(crsp.Data); err != nil {
			return err
		}

		return nil
	}

	// create new stream
	stream, err := link.Stream(ctx, creq, opts...)
	if err != nil {
		return err
	}
	defer stream.Close()

	// create client request read loop if streaming
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
	p.Links = map[string]client.Client{}
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

	// get client
	links, ok := p.Options.Values().Get("proxy.links")
	if ok {
		p.Links = links.(map[string]client.Client)
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

	go func() {
		// continuously attempt to watch routes
		for {
			// watch the routes
			p.watchRoutes()
			// in case of failure just wait a second
			time.Sleep(time.Second)
		}
	}()

	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()

		// we must refresh route metrics since they do not trigger new events
		for {
			select {
			case <-t.C:
				// refresh route metrics
				p.refreshMetrics()
			}
		}
	}()

	return p
}
