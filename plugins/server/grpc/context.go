package grpc

import (
	"context"

	"github.com/asim/go-micro/v3/server"
)

func setServerOption(k, v interface{}) server.Option {
	return func(o *server.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}
