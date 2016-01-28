package micro

import (
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/metadata"

	"golang.org/x/net/context"
)

type clientWrapper struct {
	client.Client
	headers metadata.Metadata
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	ctx = metadata.NewContext(ctx, c.headers)
	return c.Client.Call(ctx, req, rsp, opts...)
}

func (c *clientWrapper) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Streamer, error) {
	ctx = metadata.NewContext(ctx, c.headers)
	return c.Client.Stream(ctx, req, opts...)
}

func (c *clientWrapper) Publish(ctx context.Context, p client.Publication, opts ...client.PublishOption) error {
	ctx = metadata.NewContext(ctx, c.headers)
	return c.Client.Publish(ctx, p, opts...)
}
