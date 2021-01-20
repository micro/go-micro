package rabbitmq

import (
	"context"

	"github.com/asim/go-micro/v3/broker"
)

type durableQueueKey struct{}
type headersKey struct{}
type queueArgumentsKey struct{}
type prefetchCountKey struct{}
type prefetchGlobalKey struct{}
type exchangeKey struct{}
type requeueOnErrorKey struct{}
type deliveryMode struct{}
type priorityKey struct{}
type externalAuth struct{}
type durableExchange struct{}

// DurableQueue creates a durable queue when subscribing.
func DurableQueue() broker.SubscribeOption {
	return setSubscribeOption(durableQueueKey{}, true)
}

// DurableExchange is an option to set the Exchange to be durable
func DurableExchange() broker.Option {
	return setBrokerOption(durableExchange{}, true)
}

// Headers adds headers used by the headers exchange
func Headers(h map[string]interface{}) broker.SubscribeOption {
	return setSubscribeOption(headersKey{}, h)
}

// QueueArguments sets arguments for queue creation
func QueueArguments(h map[string]interface{}) broker.SubscribeOption {
	return setSubscribeOption(queueArgumentsKey{}, h)
}

// RequeueOnError calls Nack(muliple:false, requeue:true) on amqp delivery when handler returns error
func RequeueOnError() broker.SubscribeOption {
	return setSubscribeOption(requeueOnErrorKey{}, true)
}

// ExchangeName is an option to set the ExchangeName
func ExchangeName(e string) broker.Option {
	return setBrokerOption(exchangeKey{}, e)
}

// PrefetchCount ...
func PrefetchCount(c int) broker.Option {
	return setBrokerOption(prefetchCountKey{}, c)
}

// PrefetchGlobal creates a durable queue when subscribing.
func PrefetchGlobal() broker.Option {
	return setBrokerOption(prefetchGlobalKey{}, true)
}

// DeliveryMode sets a delivery mode for publishing
func DeliveryMode(value uint8) broker.PublishOption {
	return setPublishOption(deliveryMode{}, value)
}

// Priority sets a priority level for publishing
func Priority(value uint8) broker.PublishOption {
	return setPublishOption(priorityKey{}, value)
}

func ExternalAuth() broker.Option {
	return setBrokerOption(externalAuth{}, ExternalAuthentication{})
}

type subscribeContextKey struct{}

// SubscribeContext set the context for broker.SubscribeOption
func SubscribeContext(ctx context.Context) broker.SubscribeOption {
	return setSubscribeOption(subscribeContextKey{}, ctx)
}

type ackSuccessKey struct{}

// AckOnSuccess will automatically acknowledge messages when no error is returned
func AckOnSuccess() broker.SubscribeOption {
	return setSubscribeOption(ackSuccessKey{}, true)
}
