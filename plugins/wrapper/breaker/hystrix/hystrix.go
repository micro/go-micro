package hystrix

import (
	"context"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/asim/go-micro/v3/client"
)

type clientWrapper struct {
	client.Client
	filter   func(context.Context, error) bool
	fallback func(context.Context, error) error
}

func (cw *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	var err error
	herr := hystrix.DoC(ctx, req.Service()+"."+req.Endpoint(), func(c context.Context) error {
		err = cw.Client.Call(c, req, rsp, opts...)
		if cw.filter != nil {
			// custom error handling, filter errors that should not trigger circuit breaker
			if cw.filter(ctx, err) {
				return nil
			}
		}
		return err
	}, cw.fallback)
	if herr != nil {
		return herr
	}
	// return original error
	return err
}

// NewClientWrapper returns a hystrix client Wrapper.
func NewClientWrapper(opts ...Option) client.Wrapper {
	var options Options
	for _, o := range opts {
		o(&options)
	}
	return func(c client.Client) client.Client {
		return &clientWrapper{c, options.Filter, options.Fallback}
	}
}
