package rediscache

import (
	"context"

	"github.com/go-redis/redis/v8"
	"go-micro.dev/v5/cache"
)

type redisOptionsContextKey struct{}

// WithRedisOptions sets advanced options for redis.
func WithRedisOptions(options redis.UniversalOptions) cache.Option {
	return func(o *cache.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}

		o.Context = context.WithValue(o.Context, redisOptionsContextKey{}, options)
	}
}

func newUniversalClient(options cache.Options) redis.UniversalClient {
	if options.Context == nil {
		options.Context = context.Background()
	}

	opts, ok := options.Context.Value(redisOptionsContextKey{}).(redis.UniversalOptions)
	if !ok {
		addr := "redis://127.0.0.1:6379"
		if len(options.Address) > 0 {
			addr = options.Address
		}

		redisOptions, err := redis.ParseURL(addr)
		if err != nil {
			redisOptions = &redis.Options{Addr: addr}
		}

		return redis.NewClient(redisOptions)
	}

	if len(opts.Addrs) == 0 && len(options.Address) > 0 {
		opts.Addrs = []string{options.Address}
	}

	return redis.NewUniversalClient(&opts)
}
