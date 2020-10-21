package events

import (
	"time"

	"github.com/asim/go-micro/v3/store"
)

// Options which are used to configure the in-memory stream
type Options struct {
	Store store.Store
	TTL   time.Duration
}

// Option is a function which configures options
type Option func(o *Options)

// WithStore sets the underlying store to use
func WithStore(s store.Store) Option {
	return func(o *Options) {
		o.Store = s
	}
}

// WithTTL sets the default TTL
func StreamTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.TTL = ttl
	}
}

// PublishOptions contains all the options which can be provided when publishing an event
type PublishOptions struct {
	// Metadata contains any keys which can be used to query the data, for example a customer id
	Metadata map[string]string
	// Timestamp to set for the event, if the timestamp is a zero value, the current time will be used
	Timestamp time.Time
}

// PublishOption sets attributes on PublishOptions
type PublishOption func(o *PublishOptions)

// WithMetadata sets the Metadata field on PublishOptions
func WithMetadata(md map[string]string) PublishOption {
	return func(o *PublishOptions) {
		o.Metadata = md
	}
}

// WithTimestamp sets the timestamp field on PublishOptions
func WithTimestamp(t time.Time) PublishOption {
	return func(o *PublishOptions) {
		o.Timestamp = t
	}
}

// ConsumeOptions contains all the options which can be provided when subscribing to a topic
type ConsumeOptions struct {
	// Group is the name of the subscribers queue, if two subscribers have the same queue the message
	// should only be published to one of them
	Group string
	// Offset is the time from which the messages should be consumed from. If not provided then
	// the messages will be consumed starting from the moment the Subscription starts.
	Offset time.Time
	// AutoAck if true (default true), automatically acknowledges every message so it will not be redelivered.
	// If false specifies that each message need ts to be manually acknowledged by the subscriber.
	// If processing is successful the message should be ack'ed to remove the message from the stream.
	// If processing is unsuccessful the message should be nack'ed (negative acknowledgement) which will mean it will
	// remain on the stream to be processed again.
	AutoAck bool
	AckWait time.Duration
	// RetryLimit indicates number of times a message is retried
	RetryLimit int
	// CustomRetries indicates whether to use RetryLimit
	CustomRetries bool
}

// ConsumeOption sets attributes on ConsumeOptions
type ConsumeOption func(o *ConsumeOptions)

// WithGroup sets the Group fielf on ConsumeOptions to the value provided
func WithGroup(q string) ConsumeOption {
	return func(o *ConsumeOptions) {
		o.Group = q
	}
}

// WithOffset sets the Offset field on ConsumeOptions to the value provided
func WithOffset(t time.Time) ConsumeOption {
	return func(o *ConsumeOptions) {
		o.Offset = t
	}
}

// WithAutoAck sets the AutoAck field on ConsumeOptions and an ackWait duration after which if no ack is received
// the message is requeued in case auto ack is turned off
func WithAutoAck(ack bool, ackWait time.Duration) ConsumeOption {
	return func(o *ConsumeOptions) {
		o.AutoAck = ack
		o.AckWait = ackWait
	}
}

// WithRetryLimit sets the RetryLimit field on ConsumeOptions.
// Set to -1 for infinite retries (default)
func WithRetryLimit(retries int) ConsumeOption {
	return func(o *ConsumeOptions) {
		o.RetryLimit = retries
		o.CustomRetries = true
	}
}

func (s ConsumeOptions) GetRetryLimit() int {
	if !s.CustomRetries {
		return -1
	}
	return s.RetryLimit
}

// WriteOptions contains all the options which can be provided when writing an event to a store
type WriteOptions struct {
	// TTL is the duration the event should be recorded for, a zero value TTL indicates the event should
	// be stored indefinately
	TTL time.Duration
}

// WriteOption sets attributes on WriteOptions
type WriteOption func(o *WriteOptions)

// WithTTL sets the TTL attribute on WriteOptions
func WithTTL(d time.Duration) WriteOption {
	return func(o *WriteOptions) {
		o.TTL = d
	}
}

// ReadOptions contains all the options which can be provided when reading events from a store
type ReadOptions struct {
	// Limit the number of results to return
	Limit uint
	// Offset the results by this number, useful for paginated queries
	Offset uint
}

// ReadOption sets attributes on ReadOptions
type ReadOption func(o *ReadOptions)

// ReadLimit sets the limit attribute on ReadOptions
func ReadLimit(l uint) ReadOption {
	return func(o *ReadOptions) {
		o.Limit = 1
	}
}

// ReadOffset sets the offset attribute on ReadOptions
func ReadOffset(l uint) ReadOption {
	return func(o *ReadOptions) {
		o.Offset = 1
	}
}
