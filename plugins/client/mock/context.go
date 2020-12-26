package mock

import (
	"context"
)

type responseKey struct{}

func fromContext(ctx context.Context) (map[string][]MockResponse, bool) {
	r, ok := ctx.Value(responseKey{}).(map[string][]MockResponse)
	return r, ok
}

func newContext(ctx context.Context, r map[string][]MockResponse) context.Context {
	return context.WithValue(ctx, responseKey{}, r)
}
