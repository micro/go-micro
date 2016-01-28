package metadata

import (
	"golang.org/x/net/context"
)

type metaKey struct{}

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
