package gomicro

import (
	"github.com/micro/go-micro/client"
	cx "github.com/micro/go-micro/context"

	"golang.org/x/net/context"
)

type clientWrap struct {
	client.Client
	headers cx.Metadata
}

func (c *clientWrap) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	ctx = cx.WithMetadata(ctx, c.headers)
	return c.Client.Call(ctx, req, rsp, opts...)
}

func (c *clientWrap) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Streamer, error) {
	ctx = cx.WithMetadata(ctx, c.headers)
	return c.Client.Stream(ctx, req, opts...)
}

func (c *clientWrap) Publish(ctx context.Context, p client.Publication, opts ...client.PublishOption) error {
	ctx = cx.WithMetadata(ctx, c.headers)
	return c.Client.Publish(ctx, p, opts...)
}
