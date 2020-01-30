package cli

import (
	"context"

	"github.com/micro/cli/v2"
	"github.com/micro/go-micro/v2/config/source"
)

type contextKey struct{}

// Context sets the cli context
func Context(c *cli.Context) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, contextKey{}, c)
	}
}
