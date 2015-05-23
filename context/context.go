package context

import (
	"golang.org/x/net/context"
)

type key int

const (
	mdKey = key(0)
)

type MetaData map[string]string

func GetMetaData(ctx context.Context) (MetaData, bool) {
	md, ok := ctx.Value(mdKey).(MetaData)
	return md, ok
}

func WithMetaData(ctx context.Context, md MetaData) context.Context {
	return context.WithValue(ctx, mdKey, md)
}
