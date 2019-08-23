// Package grpc transparently forwards the grpc protocol using a go-micro client.
package grpc

import (
	"context"
	"io"
	"strings"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/client/grpc"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/proxy"
	"github.com/micro/go-micro/server"
)

// Proxy will transparently proxy requests to the backend.
// If no backend is specified it will call a service using the client.
// If the service matches the Name it will use the server.DefaultRouter.
type Proxy struct {
	// The proxy options
	options.Options

	// Endpoint specified the fixed endpoint to call.
	Endpoint string

	// The client to use for outbound requests
	Client client.Client
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

func (p *Proxy) SendRequest(ctx context.Context, req client.Request, rsp client.Response) error {
	return errors.InternalServerError("go.micro.proxy.grpc", "SendRequest is unsupported")
}

// ServeRequest honours the server.Proxy interface
func (p *Proxy) ServeRequest(ctx context.Context, req server.Request, rsp server.Response) error {
	// set default client
	if p.Client == nil {
		p.Client = grpc.NewClient()
	}

	opts := []client.CallOption{}

	// service name
	service := req.Service()
	endpoint := req.Endpoint()

	// call a specific backend
	if len(p.Endpoint) > 0 {
		// address:port
		if parts := strings.Split(p.Endpoint, ":"); len(parts) > 1 {
			opts = append(opts, client.WithAddress(p.Endpoint))
			// use as service name
		} else {
			service = p.Endpoint
		}
	}

	// create new request with raw bytes body
	creq := p.Client.NewRequest(service, endpoint, nil, client.WithContentType(req.ContentType()))

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
func NewProxy(opts ...options.Option) proxy.Proxy {
	p := new(Proxy)
	p.Options = options.NewOptions(opts...)
	p.Options.Init(options.WithString("grpc"))

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

	return p
}

// NewSingleHostProxy returns a router which sends requests to a single backend
func NewSingleHostProxy(url string) *Proxy {
	return &Proxy{
		Endpoint: url,
	}
}
