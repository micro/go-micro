// Package grpc transparently forwards the grpc protocol using a go-micro client.
package grpc

import (
	"context"
	"io"
	"strings"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/grpc"
	"github.com/micro/go-micro/v2/codec"
	"github.com/micro/go-micro/v2/proxy"
	"github.com/micro/go-micro/v2/server"
)

// Proxy will transparently proxy requests to the backend.
// If no backend is specified it will call a service using the client.
// If the service matches the Name it will use the server.DefaultRouter.
type Proxy struct {
	// The proxy options
	options proxy.Options

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

// ProcessMessage acts as a message exchange and forwards messages to ongoing topics
// TODO: should we look at p.Endpoint and only send to the local endpoint? probably
func (p *Proxy) ProcessMessage(ctx context.Context, msg server.Message) error {
	// TODO: check that we're not broadcast storming by sending to the same topic
	// that we're actually subscribed to

	// directly publish to the local client
	return p.Client.Publish(ctx, msg)
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
}

func (p *Proxy) String() string {
	return "grpc"
}

// NewProxy returns a new grpc proxy server
func NewProxy(opts ...proxy.Option) proxy.Proxy {
	var options proxy.Options
	for _, o := range opts {
		o(&options)
	}

	p := new(Proxy)
	p.Endpoint = options.Endpoint
	p.Client = options.Client

	return p
}

// NewSingleHostProxy returns a router which sends requests to a single backend
func NewSingleHostProxy(url string) *Proxy {
	return &Proxy{
		Endpoint: url,
	}
}
