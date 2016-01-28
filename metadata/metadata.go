package metadata

import (
	"golang.org/x/net/context"
)

type contextKeyT string

const (
	contextKey = contextKeyT("github.com/micro/go-micro/metadata")
)

type Metadata map[string]string

func FromContext(ctx context.Context) (Metadata, bool) {
	md, ok := ctx.Value(contextKey).(Metadata)
	return md, ok
}

func NewContext(ctx context.Context, md Metadata) context.Context {
	if emd, ok := ctx.Value(contextKey).(Metadata); ok {
		for k, v := range emd {
			if _, ok := md[k]; !ok {
				md[k] = v
			}
		}
	}
	return context.WithValue(ctx, contextKey, md)
}
