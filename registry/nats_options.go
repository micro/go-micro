//go:build nats
// +build nats

package registry

import (
	"context"

	"github.com/nats-io/nats.go"
)

type contextQuorumKey struct{}
type optionsKey struct{}
type watchTopicKey struct{}
type queryTopicKey struct{}
type registerActionKey struct{}

var (
	DefaultQuorum = 0
)

func getQuorum(o Options) int {
	if o.Context == nil {
		return DefaultQuorum
	}

	value := o.Context.Value(contextQuorumKey{})
	if v, ok := value.(int); ok {
		return v
	} else {
		return DefaultQuorum
	}
}

func Quorum(n int) Option {
	return func(o *Options) {
		o.Context = context.WithValue(o.Context, contextQuorumKey{}, n)
	}
}

// Options allow to inject a nats.Options struct for configuring
// the nats connection.
func NatsOptions(nopts nats.Options) Option {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, optionsKey{}, nopts)
	}
}

// QueryTopic allows to set a custom nats topic on which service registries
// query (survey) other services. All registries listen on this topic and
// then respond to the query message.
func QueryTopic(s string) Option {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, queryTopicKey{}, s)
	}
}

// WatchTopic allows to set a custom nats topic on which registries broadcast
// changes (e.g. when services are added, updated or removed). Since we don't
// have a central registry service, each service typically broadcasts in a
// determined frequency on this topic.
func WatchTopic(s string) Option {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, watchTopicKey{}, s)
	}
}

// RegisterAction allows to set the action to use when registering to nats.
// As of now there are three different options:
// - "create" (default) only registers if there is noone already registered under the same key.
// - "update" only updates the registration if it already exists.
// - "put" creates or updates a registration
func RegisterAction(s string) Option {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, registerActionKey{}, s)
	}
}
