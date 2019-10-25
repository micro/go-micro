// Package metadata is a way of defining message headers
package metadata

import (
	"context"
)

type metaKey struct{}

// Metadata is our way of representing request headers internally.
// They're used at the RPC level and translate back and forth
// from Transport headers.
type Metadata map[string]string

func Copy(md Metadata) Metadata {
	cmd := make(Metadata)
	for k, v := range md {
		cmd[k] = v
	}
	return cmd
}

func FromContext(ctx context.Context) (Metadata, bool) {
	md, ok := ctx.Value(metaKey{}).(Metadata)
	return md, ok
}

func NewContext(ctx context.Context, md Metadata) context.Context {
	return context.WithValue(ctx, metaKey{}, md)
}

// PatchContext : will add/replace source metadata fields with given patch metadata fields
func PatchContext(ctx context.Context, patchMd Metadata) context.Context {
	md, _ := ctx.Value(metaKey{}).(Metadata)
	cmd := make(Metadata)
	for k, v := range md {
		cmd[k] = v
	}
	for k, v := range patchMd {
		cmd[k] = v
	}
	return context.WithValue(ctx, metaKey{}, cmd)

}
