// Package grpc transparently forwards the grpc protocol using a go-micro client.
package grpc

import (
	"context"
	"io"
	"strings"

	"github.com/micro/go-micro"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/codec/bytes"
	"github.com/micro/go-micro/options"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-micro/service/grpc"
)

// Proxy will transparently proxy requests to the backend.
// If no backend is specified it will call a service using the client.
// If the service matches the Name it will use the server.DefaultRouter.
type Proxy struct {
	// Name of the local service. In the event it's to be left alone
	Name string

	// Backend is a single backend to route to
	// If backend is of the form address:port it will call the address.
	// Otherwise it will use it as the service name to call.
	Backend string

	// Endpoint specified the fixed endpoint to call.
	// In the event you proxy to a fixed backend this lets you
	// call a single endpoint
	Endpoint string

	// The client to use for outbound requests
	Client client.Client

	// The proxy options
	Options options.Options
}

var (
	// The default name of this local service
	DefaultName = "go.micro.proxy"
	// The default router
	DefaultProxy = &Proxy{}
)

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

// ServeRequest honours the server.Proxy interface
func (p *Proxy) ServeRequest(ctx context.Context, req server.Request, rsp server.Response) error {
	// set the default name e.g local proxy
	if p.Name == "" {
		p.Name = DefaultName
	}

	// set default client
	if p.Client == nil {
		p.Client = client.DefaultClient
	}

	// check service route
	if req.Service() == p.Name {
		// use the default router
		return server.DefaultRouter.ServeRequest(ctx, req, rsp)
	}

	opts := []client.CallOption{}

	// service name
	service := req.Service()
	endpoint := req.Endpoint()

	// call a specific backend
	if len(p.Backend) > 0 {
		// address:port
		if parts := strings.Split(p.Backend, ":"); len(parts) > 0 {
			opts = append(opts, client.WithAddress(p.Backend))
			// use as service name
		} else {
			service = p.Backend
		}
	}

	// call a specific endpoint
	if len(p.Endpoint) > 0 {
		endpoint = p.Endpoint
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

// NewProxy returns a new grpc proxy server
func NewProxy(opts ...options.Option) *Proxy {
	return &Proxy{
		Options: options.NewOptions(opts...),
	}
}

// NewSingleHostProxy returns a router which sends requests to a single backend
//
// It is used by setting it in a new micro service to act as a proxy for a backend.
//
// Usage:
//
// Create a new router to the http backend
//
// 	r := NewSingleHostProxy("localhost:10001")
//
// 	// Create your new service
// 	service := micro.NewService(
// 		micro.Name("greeter"),
//		// Set the router
//		http.WithProxy(r),
// 	)
//
// 	// Run the service
// 	service.Run()
func NewSingleHostProxy(url string) *Proxy {
	return &Proxy{
		Backend: url,
	}
}

// NewService returns a new proxy. It acts as a micro service proxy.
// Any request on the transport is routed to via the client to a service.
// In the event a backend is specified then it routes to that backend.
// The name of the backend can be a local address:port or a service name.
//
// Usage:
//
//	New micro proxy routes via micro client to any service
//
// 	proxy := NewService()
//
//	OR with address:port routes to local service
//
// 	service := NewService(
//		// Sets the default http endpoint
//		proxy.WithBackend("localhost:10001"),
//	 )
//
// 	OR with service name routes to a fixed backend service
//
// 	service := NewService(
//		// Sets the backend service
//		proxy.WithBackend("greeter"),
//	 )
//
func NewService(opts ...micro.Option) micro.Service {
	router := DefaultProxy
	name := DefaultName

	// prepend router to opts
	opts = append([]micro.Option{
		micro.Name(name),
		WithRouter(router),
	}, opts...)

	// create the new service
	service := grpc.NewService(opts...)

	// set router name
	router.Name = service.Server().Options().Name

	return service
}
