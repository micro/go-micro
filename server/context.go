package server

import (
	"time"

	"code.google.com/p/go.net/context"
)

type ctx struct{}

func (ctx *ctx) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (ctx *ctx) Done() <-chan struct{} {
	return nil
}

func (ctx *ctx) Err() error {
	return nil
}

func (ctx *ctx) Value(key interface{}) interface{} {
	return nil
}

func newContext(parent context.Context, s *serverContext) context.Context {
	return context.WithValue(parent, "serverContext", s)
}

// return server.Context
func NewContext(ctx context.Context) (Context, bool) {
	c, ok := ctx.Value("serverContext").(*serverContext)
	return c, ok
}
