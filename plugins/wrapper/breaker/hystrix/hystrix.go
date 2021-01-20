package hystrix

import (
	"github.com/afex/hystrix-go/hystrix"
	"github.com/micro/go-micro/v2/client"

	"context"
)

type clientWrapper struct {
	client.Client
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	return hystrix.Do(req.Service()+"."+req.Endpoint(), func() error {
		return c.Client.Call(ctx, req, rsp, opts...)
	}, nil)
}

// NewClientWrapper returns a hystrix client Wrapper.
func NewClientWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		return &clientWrapper{c}
	}
}
