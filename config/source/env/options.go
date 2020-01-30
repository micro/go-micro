package env

import (
	"context"

	"strings"

	"github.com/micro/go-micro/v2/config/source"
)

type strippedPrefixKey struct{}
type prefixKey struct{}

// WithStrippedPrefix sets the environment variable prefixes to scope to.
// These prefixes will be removed from the actual config entries.
func WithStrippedPrefix(p ...string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}

		o.Context = context.WithValue(o.Context, strippedPrefixKey{}, appendUnderscore(p))
	}
}

// WithPrefix sets the environment variable prefixes to scope to.
// These prefixes will not be removed. Each prefix will be considered a top level config entry.
func WithPrefix(p ...string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, prefixKey{}, appendUnderscore(p))
	}
}

func appendUnderscore(prefixes []string) []string {
	//nolint:prealloc
	var result []string
	for _, p := range prefixes {
		if !strings.HasSuffix(p, "_") {
			result = append(result, p+"_")
			continue
		}

		result = append(result, p)
	}

	return result
}
