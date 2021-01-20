package googlepubsub

import (
	"context"
	"time"

	"github.com/micro/go-micro/v2/broker"
	"google.golang.org/api/option"
)

type clientOptionKey struct{}

type projectIDKey struct{}

type maxOutstandingMessagesKey struct{}

type maxExtensionKey struct{}

type createSubscription struct{}

type deleteSubscription struct{}

// ClientOption is a broker Option which allows google pubsub client options to be
// set for the client
func ClientOption(c ...option.ClientOption) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, clientOptionKey{}, c)
	}
}

// ProjectID provides an option which sets the google project id
func ProjectID(id string) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, projectIDKey{}, id)
	}
}

// CreateSubscription prevents the creation of the subscription if it not exists
func CreateSubscription(b bool) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}

		o.Context = context.WithValue(o.Context, createSubscription{}, b)
	}
}

// DeleteSubscription prevents the deletion of the subscription if it not exists
func DeleteSubscription(b bool) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}

		o.Context = context.WithValue(o.Context, deleteSubscription{}, b)
	}
}

// MaxOutstandingMessages sets the maximum number of unprocessed messages
// (unacknowledged but not yet expired) to receive.
func MaxOutstandingMessages(max int) broker.SubscribeOption {
	return func(o *broker.SubscribeOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}

		o.Context = context.WithValue(o.Context, maxOutstandingMessagesKey{}, max)
	}
}

// MaxExtension is the maximum period for which the Subscription should
// automatically extend the ack deadline for each message.
func MaxExtension(d time.Duration) broker.SubscribeOption {
	return func(o *broker.SubscribeOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}

		o.Context = context.WithValue(o.Context, maxExtensionKey{}, d)
	}
}
