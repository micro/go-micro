package broker

type Options struct{}

type PublishOptions struct{}

type SubscribeOptions struct {
	// AutoAck defaults to true
	AutoAck bool
	// NumHandlers defaults to 1
	NumHandlers int
	// Subscribers with the same queue name
	// will create a shared subscription where each
	// receives a subset of messages.
	Queue string
}

type Option func(*Options)

type PublishOption func(*PublishOptions)

type SubscribeOption func(*SubscribeOptions)

// DisableAutoAck will disable auto acking of messages
// after they have been handled.
func DisableAutoAck() SubscribeOption {
	return func(o *SubscribeOptions) {
		o.AutoAck = false
	}
}

// NumHandlers sets the number of concurrent handlers to create
// for a subscriber.
func NumHandlers(i int) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.NumHandlers = i
	}
}

func QueueName(name string) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Queue = name
	}
}

func newSubscribeOptions(opts ...SubscribeOption) SubscribeOptions {
	opt := SubscribeOptions{
		AutoAck:     true,
		NumHandlers: 1,
	}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}
