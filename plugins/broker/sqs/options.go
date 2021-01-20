package sqs

import (
	"context"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/asim/go-micro/v3/broker"
)

type sqsClientKey struct{}
type dedupFunctionKey struct{}
type groupIdFunctionKey struct{}
type maxMessagesKey struct{}
type visiblityTimeoutKey struct{}
type waitTimeSecondsKey struct{}

type StringFromMessageFunc func(m *broker.Message) string

// DeduplicationFunction sets the function used to create the deduplication string
// for a given message
func DeduplicationFunction(dedup StringFromMessageFunc) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, dedupFunctionKey{}, dedup)
	}
}

// GroupIDFunction sets the function used to create the group ID string for a
// given message
func GroupIDFunction(groupfunc StringFromMessageFunc) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, groupIdFunctionKey{}, groupfunc)
	}
}

// MaxReceiveMessages indicates how many messages a receive operation should pull
// during any single call
func MaxReceiveMessages(max int64) broker.SubscribeOption {
	return func(o *broker.SubscribeOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, maxMessagesKey{}, max)
	}
}

// VisibilityTimeout controls how long a message is hidden from other queue consumers
// before being put back. If a consumer does not delete the message, it will be put back
// even if it was "processed"
func VisibilityTimeout(seconds int64) broker.SubscribeOption {
	return func(o *broker.SubscribeOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, visiblityTimeoutKey{}, seconds)
	}
}

// WaitTimeSeconds controls the length of long polling for available messages
func WaitTimeSeconds(seconds int64) broker.SubscribeOption {
	return func(o *broker.SubscribeOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, waitTimeSecondsKey{}, seconds)
	}
}

// Client receives an instantiated instance of an SQS client which is used instead of initialising a new client
func Client(c *sqs.SQS) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, sqsClientKey{}, c)
	}
}
