package hystrix

import (
	"context"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/asim/go-micro/v3/client"
)

type clientWrapper struct {
	client.Client
	fallback func(error) error
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	return hystrix.Do(req.Service()+"."+req.Endpoint(), func() error {
		return c.Client.Call(ctx, req, rsp, opts...)
	}, c.fallback)
}

// NewClientWrapper returns a hystrix client Wrapper.
func NewClientWrapper(fallbacks ...func(error) error) client.Wrapper {
	return func(c client.Client) client.Client {
		return &clientWrapper{c, resolveFallback(fallbacks)}
	}
}

func resolveFallback(fallbacks []func(error) error) func(error) error {
	switch len(fallbacks) {
	case 0:
		return nil
	case 1:
		return fallbacks[0]
	default:
		panic("too many fallback parameters")
	}
}
