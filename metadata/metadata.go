// Package metadata is a way of defining message headers
package metadata

import (
	"context"
	"encoding/base32"
	"strings"
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

// Set add key with val to metadata
func Set(ctx context.Context, k, v string) context.Context {
	md, ok := FromContext(ctx)
	if !ok {
		md = make(Metadata)
	}
	md[base32.StdEncoding.EncodeToString([]byte(k))] = v
	return context.WithValue(ctx, metaKey{}, md)
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
	if !ok {
		return nil, ok
	}

	newMD := make(Metadata)

	for k, v := range md {
		key, err := base32.StdEncoding.DecodeString(strings.ToUpper(k))
		if err != nil {
			// try to be compatible with older micro versions
			newMD[strings.Title(k)] = v
			continue
		}
		newMD[string(key)] = v
	}

	return newMD, ok
}

// NewContext creates a new context with the given metadata
func NewContext(ctx context.Context, md Metadata) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	newMD := make(Metadata)
	for k, v := range md {
		newMD[base32.StdEncoding.EncodeToString([]byte(k))] = v
	}
	return context.WithValue(ctx, metaKey{}, newMD)
}

// MergeContext merges metadata to existing metadata, overwriting if specified
func MergeContext(ctx context.Context, patchMd Metadata, overwrite bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	md, _ := ctx.Value(metaKey{}).(Metadata)
	cmd := make(Metadata)
	for k, v := range md {
		cmd[k] = v
	}
	for k, v := range patchMd {
		nk := base32.StdEncoding.EncodeToString([]byte(k))
		if _, ok := cmd[nk]; ok && !overwrite {
			// skip
		} else {
			cmd[nk] = v
		}
	}
	return context.WithValue(ctx, metaKey{}, cmd)
}
