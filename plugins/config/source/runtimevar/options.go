package runtimevar

import (
	"context"

	"github.com/asim/go-micro/v3/config/source"
	"gocloud.dev/runtimevar"
)

type variableKey struct{}

// WithVariable sets the runtimevar.Variable.
func WithVariable(v *runtimevar.Variable) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, variableKey{}, v)
	}
}
