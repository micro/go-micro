package client

import (
	"context"

	example "github.com/micro/examples/template/srv/proto/example"
	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/server"
)

type exampleKey struct{}

// FromContext retrieves the client from the Context
func ExampleFromContext(ctx context.Context) (example.ExampleService, bool) {
	c, ok := ctx.Value(exampleKey{}).(example.ExampleService)
	return c, ok
}

// Client returns a wrapper for the ExampleClient
func ExampleWrapper(service micro.Service) server.HandlerWrapper {
	client := example.NewExampleService("go.micro.srv.template", service.Client())

	return func(fn server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			ctx = context.WithValue(ctx, exampleKey{}, client)
			return fn(ctx, req, rsp)
		}
	}
}
