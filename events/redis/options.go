package stream

import (
	"crypto/tls"

	"github.com/redis/go-redis/v9"
)

// Options which are used to configure the redis stream.
type Options struct {
	Address   string
	User      string
	Password  string
	TLSConfig *tls.Config

	RedisOptions *redis.UniversalOptions
}

// Option is a function which configures options.
type Option func(o *Options)

// Address sets the Redis address option.
// Needs to be a full URL with scheme (redis://, rediss://, unix://).
// (eg. redis://user:password@localhost:6789/3?dial_timeout=3).
// Alternatively, the address can simply be the `host:port` format
// where User, Password, TLSConfig are defined with their respective options.
func Address(addr string) Option {
	return func(o *Options) {
		o.Address = addr
	}
}

func User(user string) Option {
	return func(o *Options) {
		o.User = user
	}
}

func Password(password string) Option {
	return func(o *Options) {
		o.Password = password
	}
}

func TLSConfig(tlsConfig *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = tlsConfig
	}
}

// WithRedisOptions sets advanced options for redis.
func WithRedisOptions(options *redis.UniversalOptions) Option {
	return func(o *Options) {
		o.RedisOptions = options
	}
}

func (o Options) newUniversalClient() redis.UniversalClient {
	opts := o.RedisOptions

	if opts == nil {
		addr := "redis://127.0.0.1:6379"
		if len(o.Address) > 0 {
			addr = o.Address
		}

		redisOptions, err := redis.ParseURL(addr)
		if err != nil {
			redisOptions = &redis.Options{
				Addr:      addr,
				Username:  o.User,
				Password:  o.Password,
				TLSConfig: o.TLSConfig,
			}
		}

		return redis.NewClient(redisOptions)
	}

	if len(opts.Addrs) == 0 && len(o.Address) > 0 {
		opts.Addrs = []string{o.Address}
	}

	if len(opts.Username) == 0 && len(o.User) > 0 {
		opts.Username = o.User
	}

	if len(opts.Password) == 0 && len(o.Password) > 0 {
		opts.Password = o.Password
	}

	if opts.TLSConfig == nil && o.TLSConfig != nil {
		opts.TLSConfig = o.TLSConfig
	}

	return redis.NewUniversalClient(opts)
}
