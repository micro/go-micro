package ratelimit

import (
	"time"

	"github.com/juju/ratelimit"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/server"

	"context"
)

type clientWrapper struct {
	fn func() error
	client.Client
}

func limit(b *ratelimit.Bucket, wait bool, errId string) func() error {
	return func() error {
		if wait {
			time.Sleep(b.Take(1))
		} else if b.TakeAvailable(1) == 0 {
			return errors.New(errId, "too many request", 429)
		}
		return nil
	}
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	if err := c.fn(); err != nil {
		return err
	}
	return c.Client.Call(ctx, req, rsp, opts...)
}

// NewClientWrapper takes a rate limiter and wait flag and returns a client Wrapper.
func NewClientWrapper(b *ratelimit.Bucket, wait bool) client.Wrapper {
	fn := limit(b, wait, "go.micro.client")

	return func(c client.Client) client.Client {
		return &clientWrapper{fn, c}
	}
}

// NewHandlerWrapper takes a rate limiter and wait flag and returns a client Wrapper.
func NewHandlerWrapper(b *ratelimit.Bucket, wait bool) server.HandlerWrapper {
	fn := limit(b, wait, "go.micro.server")

	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			if err := fn(); err != nil {
				return err
			}
			return h(ctx, req, rsp)
		}
	}
}
