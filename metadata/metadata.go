// Package metadata is a way of defining message headers
package metadata

import (
	"context"
	"strings"
)

type MetadataKey struct{}

// Metadata is our way of representing request headers internally.
// They're used at the RPC level and translate back and forth
// from Transport headers.
type Metadata map[string]string

func (md Metadata) Get(key string) (string, bool) {
	// attempt to get as is
	val, ok := md[key]
	if ok {
		return val, ok
	}

	// attempt to get lower case
	val, ok = md[strings.Title(key)]
	return val, ok
}

func (md Metadata) Delete(key string) {
	// delete key as-is
	delete(md, key)
	// delete also Title key
	delete(md, strings.Title(key))
}

// Copy makes a copy of the metadata
func Copy(md Metadata) Metadata {
	cmd := make(Metadata)
	for k, v := range md {
		cmd[k] = v
	}
	return cmd
}

// Delete key from metadata
func Delete(ctx context.Context, k string) context.Context {
	return Set(ctx, k, "")
}

// Set add key with val to metadata
func Set(ctx context.Context, k, v string) context.Context {
	md, ok := FromContext(ctx)
	if !ok {
		md = make(Metadata)
	}
	if v == "" {
		delete(md, k)
	} else {
		md[k] = v
	}
	return context.WithValue(ctx, MetadataKey{}, md)
}

// Get returns a single value from metadata in the context
func Get(ctx context.Context, key string) (string, bool) {
	md, ok := FromContext(ctx)
	if !ok {
		return "", ok
	}
	// attempt to get as is
	val, ok := md[key]
	if ok {
		return val, ok
	}

	// attempt to get lower case
	val, ok = md[strings.Title(key)]

	return val, ok
}

// FromContext returns metadata from the given context
func FromContext(ctx context.Context) (Metadata, bool) {
	md, ok := ctx.Value(MetadataKey{}).(Metadata)
	if !ok {
		return nil, ok
	}

	// capitalise all values
	newMD := make(map[string]string, len(md))
	for k, v := range md {
		newMD[strings.Title(k)] = v
	}

	return newMD, ok
}

// NewContext creates a new context with the given metadata
func NewContext(ctx context.Context, md Metadata) context.Context {
	return context.WithValue(ctx, MetadataKey{}, md)
}

// MergeContext merges metadata to existing metadata, overwriting if specified
func MergeContext(ctx context.Context, patchMd Metadata, overwrite bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	md, _ := ctx.Value(MetadataKey{}).(Metadata)
	cmd := make(Metadata)
	for k, v := range md {
		cmd[k] = v
	}
	for k, v := range patchMd {
		if _, ok := cmd[k]; ok && !overwrite {
			// skip
		} else if v != "" {
			cmd[k] = v
		} else {
			delete(cmd, k)
		}
	}
	return context.WithValue(ctx, MetadataKey{}, cmd)
}
