package cloudflare

import (
	"context"

	"github.com/micro/go-micro/store"
)

func getOption(ctx context.Context, key string) string {
	if ctx == nil {
		return ""
	}
	val, ok := ctx.Value(key).(string)
	if !ok {
		return ""
	}
	return val
}

func getToken(ctx context.Context) string {
	return getOption(ctx, "CF_API_TOKEN")
}

func getAccount(ctx context.Context) string {
	return getOption(ctx, "CF_ACCOUNT_ID")
}

func getNamespace(ctx context.Context) string {
	return getOption(ctx, "KV_NAMESPACE_ID")
}

// Token sets the cloudflare api token
func Token(t string) store.Option {
	return func(o *store.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, "CF_API_TOKEN", t)
	}
}

// Account sets the cloudflare account id
func Account(id string) store.Option {
	return func(o *store.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, "CF_ACCOUNT_ID", id)
	}
}

// Namespace sets the KV namespace
func Namespace(ns string) store.Option {
	return func(o *store.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, "KV_NAMESPACE_ID", ns)
	}
}
