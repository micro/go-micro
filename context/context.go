package context

import (
	"golang.org/x/net/context"
)

type key int

const (
	mdKey = key(0)
)

type Metadata map[string]string

func GetMetadata(ctx context.Context) (Metadata, bool) {
	md, ok := ctx.Value(mdKey).(Metadata)
	return md, ok
}

func WithMetadata(ctx context.Context, md Metadata) context.Context {
	if emd, ok := ctx.Value(mdKey).(Metadata); ok {
		for k, v := range emd {
			if _, ok := md[k]; !ok {
				md[k] = v
			}
		}
	}
	return context.WithValue(ctx, mdKey, md)
}
