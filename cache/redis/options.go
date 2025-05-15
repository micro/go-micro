package redis

import (
	"context"

	rclient "github.com/go-redis/redis/v8"
	"go-micro.dev/v5/cache"
)

type redisOptionsContextKey struct{}

// WithRedisOptions sets advanced options for redis.
func WithRedisOptions(options rclient.UniversalOptions) cache.Option {
	return func(o *cache.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}

		o.Context = context.WithValue(o.Context, redisOptionsContextKey{}, options)
	}
}

func newUniversalClient(options cache.Options) rclient.UniversalClient {
	if options.Context == nil {
		options.Context = context.Background()
	}

	opts, ok := options.Context.Value(redisOptionsContextKey{}).(rclient.UniversalOptions)
	if !ok {
		addr := "redis://127.0.0.1:6379"
		if len(options.Address) > 0 {
			addr = options.Address
		}

		redisOptions, err := rclient.ParseURL(addr)
		if err != nil {
			redisOptions = &rclient.Options{Addr: addr}
		}

		return rclient.NewClient(redisOptions)
	}

	if len(opts.Addrs) == 0 && len(options.Address) > 0 {
		opts.Addrs = []string{options.Address}
	}

	return rclient.NewUniversalClient(&opts)
}
