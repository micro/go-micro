// Package wrapper injects a go-micro.Service into the context
package service

import (
	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/server"

	"context"
)

type clientWrapper struct {
	service micro.Service
	client.Client
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	ctx = micro.NewContext(ctx, c.service)
	return c.Client.Call(ctx, req, rsp, opts...)
}

// NewClientWrapper wraps a service within a client so it can be accessed by subsequent client wrappers.
func NewClientWrapper(service micro.Service) client.Wrapper {
	return func(c client.Client) client.Client {
		return &clientWrapper{service, c}
	}
}

// NewHandlerWrapper wraps a service within the handler so it can be accessed by the handler itself.
func NewHandlerWrapper(service micro.Service) server.HandlerWrapper {
	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			ctx = micro.NewContext(ctx, service)
			return h(ctx, req, rsp)
		}
	}
}
