package mock

import (
	"context"
)

type responseKey struct{}

func responseFromContext(ctx context.Context) (map[string][]MockResponse, bool) {
	r, ok := ctx.Value(responseKey{}).(map[string][]MockResponse)
	return r, ok
}

func newResponseContext(ctx context.Context, r map[string][]MockResponse) context.Context {
	return context.WithValue(ctx, responseKey{}, r)
}

type subscriberKey struct{}

func subscriberFromContext(ctx context.Context) (map[string]MockSubscriber, bool) {
	r, ok := ctx.Value(subscriberKey{}).(map[string]MockSubscriber)
	return r, ok
}

func newSubscriberContext(ctx context.Context, r map[string]MockSubscriber) context.Context {
	return context.WithValue(ctx, subscriberKey{}, r)
}
