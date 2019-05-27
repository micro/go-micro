package server

import (
	"context"
	"sync"
)

type serverKey struct{}

func wait(ctx context.Context) *sync.WaitGroup {
	if ctx == nil {
		return nil
	}
	wg, ok := ctx.Value("wait").(*sync.WaitGroup)
	if !ok {
		return nil
	}
	return wg
}

func FromContext(ctx context.Context) (Server, bool) {
	c, ok := ctx.Value(serverKey{}).(Server)
	return c, ok
}

func NewContext(ctx context.Context, s Server) context.Context {
	return context.WithValue(ctx, serverKey{}, s)
}
