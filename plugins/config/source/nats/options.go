package nats

import (
	"context"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"go-micro.dev/v4/config/source"
)

type (
	urlKey    struct{}
	bucketKey struct{}
	keyKey    struct{}
)

// WithUrl sets the nats url
func WithUrl(a string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, urlKey{}, a)
	}
}

// WithBucket sets the nats key
func WithBucket(a string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, bucketKey{}, a)
	}
}

// WithKey sets the nats key
func WithKey(a string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, keyKey{}, a)
	}
}

func Client(url string) (natsgo.JetStreamContext, error) {
	nc, err := natsgo.Connect(url)
	if err != nil {
		return nil, err
	}

	return nc.JetStream(natsgo.MaxWait(10 * time.Second))
}
