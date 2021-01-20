package pkger

import (
	"context"

	"github.com/asim/go-micro/v3/config/source"
)

type pkgerPathKey struct{}

// WithPath sets the path to pkger
func WithPath(p string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, pkgerPathKey{}, p)
	}
}
