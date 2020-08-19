package file

import "context"

type Options struct {
	Context context.Context
}

type Option func(o *Options)

func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}
