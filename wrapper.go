package micro

import (
	"github.com/micro/go-micro/client"
	cx "github.com/micro/go-micro/context"

	"golang.org/x/net/context"
)

type clientWrapper struct {
	client.Client
	headers cx.Metadata
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	ctx = cx.WithMetadata(ctx, c.headers)
	return c.Client.Call(ctx, req, rsp, opts...)
}

func (c *clientWrapper) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Streamer, error) {
	ctx = cx.WithMetadata(ctx, c.headers)
	return c.Client.Stream(ctx, req, opts...)
}

func (c *clientWrapper) Publish(ctx context.Context, p client.Publication, opts ...client.PublishOption) error {
	ctx = cx.WithMetadata(ctx, c.headers)
	return c.Client.Publish(ctx, p, opts...)
}
