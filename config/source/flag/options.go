package flag

import (
	"context"

	"github.com/micro/go-micro/v2/config/source"
)

type includeUnsetKey struct{}

// IncludeUnset toggles the loading of unset flags and their respective default values.
// Default behavior is to ignore any unset flags.
func IncludeUnset(b bool) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, includeUnsetKey{}, true)
	}
}
