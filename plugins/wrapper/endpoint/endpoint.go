// Package endpoint provides a wrapper that executes other wrappers for specific methods
package endpoint

import (
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/server"

	"context"
)

type clientWrapper struct {
	endpoints map[string]bool // endpoints to execute on
	wrapper   client.Wrapper  // the original wrapper
	client.Client
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	// execute client wrapper?
	if !c.endpoints[req.Endpoint()] {
		// no
		return c.Client.Call(ctx, req, rsp, opts...)
	}

	// yes
	return c.wrapper(c.Client).Call(ctx, req, rsp, opts...)
}

// NewClientWrapper wraps another client wrapper but only executes when the endpoints specified are executed.
func NewClientWrapper(cw client.Wrapper, eps ...string) client.Wrapper {
	// create map of endpoints
	endpoints := make(map[string]bool)
	for _, ep := range eps {
		endpoints[ep] = true
	}

	return func(c client.Client) client.Client {
		return &clientWrapper{endpoints, cw, c}
	}
}

// NewHandlerWrapper wraps another handler wrapper but only executes when the endpoints specified are executed.
func NewHandlerWrapper(hw server.HandlerWrapper, eps ...string) server.HandlerWrapper {
	// create map of endpoints
	endpoints := make(map[string]bool)
	for _, ep := range eps {
		endpoints[ep] = true
	}

	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			// execute the handler wrapper?
			if !endpoints[req.Endpoint()] {
				// no
				return h(ctx, req, rsp)
			}

			// yes
			return hw(h)(ctx, req, rsp)
		}
	}
}
