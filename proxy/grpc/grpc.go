// Package grpc is a grpc proxy built for the go-micro/server
package grpc

import (
	"context"
	"io"
	"strings"

	"github.com/micro/go-micro/v3/client"
	grpcc "github.com/micro/go-micro/v3/client/grpc"
	"github.com/micro/go-micro/v3/codec"
	"github.com/micro/go-micro/v3/codec/bytes"
	"github.com/micro/go-micro/v3/errors"
	"github.com/micro/go-micro/v3/logger"
	"github.com/micro/go-micro/v3/proxy"
	"github.com/micro/go-micro/v3/server"
	"google.golang.org/grpc"
)

// Proxy will transparently proxy requests to an endpoint.
// If no endpoint is specified it will call a service using the client.
type Proxy struct {
	// embed options
	options proxy.Options

	// The client to use for outbound requests in the local network
	Client client.Client

	// Endpoint to route all calls to
	Endpoint string
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

	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("Proxy received message for %s", msg.Topic())
	}

	// directly publish to the local client
	return p.Client.Publish(ctx, msg)
}

// ServeRequest honours the server.Router interface
func (p *Proxy) ServeRequest(ctx context.Context, req server.Request, rsp server.Response) error {
	// service name to call
	service := req.Service()
	// endpoint to call
	endpoint := req.Endpoint()

	if len(service) == 0 {
		return errors.BadRequest("go.micro.proxy", "service name is blank")
	}

	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("Proxy received request for %s %s", service, endpoint)
	}

	// no retries with the proxy
	opts := []client.CallOption{
		client.WithRetries(0),
	}

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

	// serve the normal way
	return p.serveRequest(ctx, p.Client, service, endpoint, req, rsp, opts...)
}

func (p *Proxy) serveRequest(ctx context.Context, link client.Client, service, endpoint string, req server.Request, rsp server.Response, opts ...client.CallOption) error {
	// read initial request
	body, err := req.Read()
	if err != nil {
		return err
	}

	// create new request with raw bytes body
	creq := link.NewRequest(service, endpoint, &bytes.Frame{Data: body}, client.WithContentType(req.ContentType()))

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

	// new context with cancel
	ctx, cancel := context.WithCancel(ctx)

	// create new stream
	stream, err := link.Stream(ctx, creq, opts...)
	if err != nil {
		return err
	}
	defer stream.Close()

	// with a grpc stream we have to refire the initial request
	// client request to start the server side

	// get the header from client
	msg := &codec.Message{
		Type:   codec.Request,
		Header: req.Header(),
		Body:   body,
	}

	// write the raw request
	err = stream.Request().Codec().Write(msg, nil)
	if err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}

	// create client request read loop if streaming
	go func() {
		err := readLoop(req, stream)
		if err != nil && err != io.EOF {
			// cancel the context
			cancel()
		}
	}()

	// get raw response
	resp := stream.Response()

	// create server response write loop
	for {
		// read backend response body
		body, err := resp.Read()
		if err != nil {
			// when we're done if its a grpc stream we have to set the trailer
			if cc, ok := stream.(grpc.ClientStream); ok {
				if ss, ok := resp.Codec().(grpc.ServerStream); ok {
					ss.SetTrailer(cc.Trailer())
				}
			}
		}

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

// NewProxy returns a new proxy which will route based on mucp headers
func NewProxy(opts ...proxy.Option) proxy.Proxy {
	var options proxy.Options

	for _, o := range opts {
		o(&options)
	}

	// create a new grpc proxy
	p := new(Proxy)
	p.options = options

	// set the client
	p.Client = options.Client
	// set the endpoint
	p.Endpoint = options.Endpoint

	// set the default client
	if p.Client == nil {
		p.Client = grpcc.NewClient()
	}

	return p
}
