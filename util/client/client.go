package client

import (
	"context"

	"github.com/micro/go-micro/v3/client"
)

type staticClient struct {
	address string
	client.Client
}

func (s *staticClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	return s.Client.Call(ctx, req, rsp, append(opts, client.WithAddress(s.address))...)
}

func (s *staticClient) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	return s.Client.Stream(ctx, req, append(opts, client.WithAddress(s.address))...)
}

// StaticClient sets an address on every call
func Static(address string, c client.Client) client.Client {
	return &staticClient{address, c}
}
