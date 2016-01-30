package metadata

import (
	"golang.org/x/net/context"
)

type metaKey struct{}

// Metadata is our way of representing request headers internally.
// They're used at the RPC level and translate back and forth
// from Transport headers.
type Metadata map[string]string

func FromContext(ctx context.Context) (Metadata, bool) {
	md, ok := ctx.Value(metaKey{}).(Metadata)
	return md, ok
}

func NewContext(ctx context.Context, md Metadata) context.Context {
	if emd, ok := ctx.Value(metaKey{}).(Metadata); ok {
		for k, v := range emd {
			if _, ok := md[k]; !ok {
				md[k] = v
			}
		}
	}
	return context.WithValue(ctx, metaKey{}, md)
}
