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

// Copy makes a copy of the metadata
func Copy(md Metadata) Metadata {
	cmd := make(Metadata)
	for k, v := range md {
		cmd[k] = v
	}
	return cmd
}

// Get returns a single value from metadata in the context
func Get(ctx context.Context, key string) (string, bool) {
	md, ok := FromContext(ctx)
	if !ok {
		return "", ok
	}
	val, ok := md[key]
	return val, ok
}

// FromContext returns metadata from the given context
func FromContext(ctx context.Context) (Metadata, bool) {
	md, ok := ctx.Value(metaKey{}).(Metadata)
	return md, ok
}

// NewContext creates a new context with the given metadata
func NewContext(ctx context.Context, md Metadata) context.Context {
	return context.WithValue(ctx, metaKey{}, md)
}

// MergeContext merges metadata to existing metadata, overwriting if specified
func MergeContext(ctx context.Context, patchMd Metadata, overwrite bool) context.Context {
	md, _ := ctx.Value(metaKey{}).(Metadata)
	cmd := make(Metadata)
	for k, v := range md {
		cmd[k] = v
	}
	for k, v := range patchMd {
		if _, ok := cmd[k]; ok && !overwrite {
			// skip
		} else {
			cmd[k] = v
		}
	}
	return context.WithValue(ctx, metaKey{}, cmd)

}
