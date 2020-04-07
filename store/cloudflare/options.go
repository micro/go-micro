package cloudflare

import (
	"context"
	"time"

	"github.com/micro/go-micro/v2/store"
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
		o.Database = ns
	}
}

// CacheTTL sets the timeout in nanoseconds of the read/write cache
func CacheTTL(ttl time.Duration) store.Option {
	return func(o *store.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, "STORE_CACHE_TTL", ttl)
	}
}
