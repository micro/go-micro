package client

import (
	"context"
)

type clientKey struct{}

func FromContext(ctx context.Context) (Client, bool) {
	c, ok := ctx.Value(clientKey{}).(Client)
	return c, ok
}

func NewContext(ctx context.Context, c Client) context.Context {
	return context.WithValue(ctx, clientKey{}, c)
}
