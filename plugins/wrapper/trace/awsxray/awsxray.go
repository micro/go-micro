// Package awsxray is a wrapper for AWS X-Ray distributed tracing
package awsxray

import (
	"context"
	"github.com/asim/go-awsxray"
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/server"
)

type xrayWrapper struct {
	opts Options
	x    *awsxray.AWSXRay
	client.Client
}

func (x *xrayWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	var err error
	s := getSegment(x.opts.Name, ctx)

	defer func() {
		setCallStatus(s, req.Service(), req.Endpoint(), err)
		go record(x.x, s)
	}()

	ctx = newContext(ctx, s)
	err = x.Client.Call(ctx, req, rsp, opts...)
	return err
}

// NewCallWrapper accepts Options and returns a Trace Call Wrapper for individual node calls made by the client
func NewCallWrapper(opts ...Option) client.CallWrapper {
	options := Options{
		Name:   "go.micro.client.CallFunc",
		Daemon: "localhost:2000",
	}

	for _, o := range opts {
		o(&options)
	}

	x := newXRay(options)

	return func(cf client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
			var err error
			s := getSegment(options.Name, ctx)

			defer func() {
				setCallStatus(s, node.Address, req.Endpoint(), err)
				go record(x, s)
			}()

			ctx = newContext(ctx, s)
			err = cf(ctx, node, req, rsp, opts)
			return err
		}
	}
}

// NewClientWrapper accepts Options and returns a Trace Client Wrapper which tracks high level service calls
func NewClientWrapper(opts ...Option) client.Wrapper {
	options := Options{
		Name:   "go.micro.client.Call",
		Daemon: "localhost:2000",
	}

	for _, o := range opts {
		o(&options)
	}

	return func(c client.Client) client.Client {
		return &xrayWrapper{options, newXRay(options), c}
	}
}

// NewHandlerWrapper accepts Options and returns a Trace Handler Wrapper
func NewHandlerWrapper(opts ...Option) server.HandlerWrapper {
	options := Options{
		Daemon: "localhost:2000",
	}

	for _, o := range opts {
		o(&options)
	}

	x := newXRay(options)

	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			name := options.Name
			if len(name) == 0 {
				// default name
				name = req.Service() + "." + req.Endpoint()
			}

			var err error
			s := getSegment(name, ctx)

			defer func() {
				setCallStatus(s, req.Service(), req.Endpoint(), err)
				go record(x, s)
			}()

			ctx = newContext(ctx, s)
			err = h(ctx, req, rsp)
			return err
		}
	}
}
