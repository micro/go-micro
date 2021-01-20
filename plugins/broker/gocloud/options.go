package gocloud

import (
	"context"

	"github.com/micro/go-micro/v2/broker"
	"gocloud.dev/gcp"
)

type (
	rabbitURLKey      struct{}
	gcpTokenSourceKey struct{}
	gcpProjectIDKey   struct{}
)

// RabbitURL is a broker Option that provides a URL for
// Go Cloud's RabbitMQ implementation.
func RabbitURL(url string) broker.Option {
	return optfunc(rabbitURLKey{}, url)
}

// GCPTokenSource is a broker Option that provides a TokenSource
// for Go Cloud's Google Pub/Sub implementation.
func GCPTokenSource(ts gcp.TokenSource) broker.Option {
	return optfunc(gcpTokenSourceKey{}, ts)
}

// GCPProjectID is a broker Option that provides a project ID
// for Go Cloud's Google Pub/Sub implementation.
func GCPProjectID(projID gcp.ProjectID) broker.Option {
	return optfunc(gcpProjectIDKey{}, projID)
}

func optfunc(key, val interface{}) func(*broker.Options) {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, key, val)
	}
}
