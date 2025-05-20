package rabbitmq

import (
	"context"
	"time"

	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/server"
)

type durableQueueKey struct{}
type headersKey struct{}
type queueArgumentsKey struct{}
type prefetchCountKey struct{}
type prefetchGlobalKey struct{}
type confirmPublishKey struct{}
type exchangeKey struct{}
type exchangeTypeKey struct{}
type withoutExchangeKey struct{}
type requeueOnErrorKey struct{}
type deliveryMode struct{}
type priorityKey struct{}
type contentType struct{}
type contentEncoding struct{}
type correlationID struct{}
type replyTo struct{}
type expiration struct{}
type messageID struct{}
type timestamp struct{}
type typeMsg struct{}
type userID struct{}
type appID struct{}
type externalAuth struct{}
type durableExchange struct{}

// ServerDurableQueue provide durable queue option for micro.RegisterSubscriber
func ServerDurableQueue() server.SubscriberOption {
	return setServerSubscriberOption(durableQueueKey{}, true)
}

// ServerAckOnSuccess export AckOnSuccess server.SubscriberOption
func ServerAckOnSuccess() server.SubscriberOption {
	return setServerSubscriberOption(ackSuccessKey{}, true)
}

// DurableQueue creates a durable queue when subscribing.
func DurableQueue() broker.SubscribeOption {
	return setSubscribeOption(durableQueueKey{}, true)
}

// DurableExchange is an option to set the Exchange to be durable.
func DurableExchange() broker.Option {
	return setBrokerOption(durableExchange{}, true)
}

// Headers adds headers used by the headers exchange.
func Headers(h map[string]interface{}) broker.SubscribeOption {
	return setSubscribeOption(headersKey{}, h)
}

// QueueArguments sets arguments for queue creation.
func QueueArguments(h map[string]interface{}) broker.SubscribeOption {
	return setSubscribeOption(queueArgumentsKey{}, h)
}

func RequeueOnError() broker.SubscribeOption {
	return setSubscribeOption(requeueOnErrorKey{}, true)
}

// ExchangeName is an option to set the ExchangeName.
func ExchangeName(e string) broker.Option {
	return setBrokerOption(exchangeKey{}, e)
}

// WithoutExchange is an option to use the rabbitmq default exchange.
// means it would not create any custom exchange.
func WithoutExchange() broker.Option {
	return setBrokerOption(withoutExchangeKey{}, true)
}

// ExchangeType is an option to set the rabbitmq exchange type.
func ExchangeType(t MQExchangeType) broker.Option {
	return setBrokerOption(exchangeTypeKey{}, t)
}

// PrefetchCount ...
func PrefetchCount(c int) broker.Option {
	return setBrokerOption(prefetchCountKey{}, c)
}

// PrefetchGlobal creates a durable queue when subscribing.
func PrefetchGlobal() broker.Option {
	return setBrokerOption(prefetchGlobalKey{}, true)
}

// ConfirmPublish ensures all published messages are confirmed by waiting for an ack/nack from the broker.
func ConfirmPublish() broker.Option {
	return setBrokerOption(confirmPublishKey{}, true)
}

// DeliveryMode sets a delivery mode for publishing.
func DeliveryMode(value uint8) broker.PublishOption {
	return setPublishOption(deliveryMode{}, value)
}

// Priority sets a priority level for publishing.
func Priority(value uint8) broker.PublishOption {
	return setPublishOption(priorityKey{}, value)
}

// ContentType sets a property MIME content type for publishing.
func ContentType(value string) broker.PublishOption {
	return setPublishOption(contentType{}, value)
}

// ContentEncoding sets a property MIME content encoding for publishing.
func ContentEncoding(value string) broker.PublishOption {
	return setPublishOption(contentEncoding{}, value)
}

// CorrelationID sets a property correlation ID for publishing.
func CorrelationID(value string) broker.PublishOption {
	return setPublishOption(correlationID{}, value)
}

// ReplyTo sets a property address to to reply to (ex: RPC) for publishing.
func ReplyTo(value string) broker.PublishOption {
	return setPublishOption(replyTo{}, value)
}

// Expiration sets a property message expiration spec for publishing.
func Expiration(value string) broker.PublishOption {
	return setPublishOption(expiration{}, value)
}

// MessageId sets a property message identifier for publishing.
func MessageId(value string) broker.PublishOption {
	return setPublishOption(messageID{}, value)
}

// Timestamp sets a property message timestamp for publishing.
func Timestamp(value time.Time) broker.PublishOption {
	return setPublishOption(timestamp{}, value)
}

// TypeMsg sets a property message type name for publishing.
func TypeMsg(value string) broker.PublishOption {
	return setPublishOption(typeMsg{}, value)
}

// UserID sets a property user id for publishing.
func UserID(value string) broker.PublishOption {
	return setPublishOption(userID{}, value)
}

// AppID sets a property application id for publishing.
func AppID(value string) broker.PublishOption {
	return setPublishOption(appID{}, value)
}

func ExternalAuth() broker.Option {
	return setBrokerOption(externalAuth{}, ExternalAuthentication{})
}

type subscribeContextKey struct{}

// SubscribeContext set the context for broker.SubscribeOption.
func SubscribeContext(ctx context.Context) broker.SubscribeOption {
	return setSubscribeOption(subscribeContextKey{}, ctx)
}

type ackSuccessKey struct{}

// AckOnSuccess will automatically acknowledge messages when no error is returned.
func AckOnSuccess() broker.SubscribeOption {
	return setSubscribeOption(ackSuccessKey{}, true)
}

// PublishDeliveryMode client.PublishOption for setting message "delivery mode"
// mode , Transient (0 or 1) or Persistent (2)
func PublishDeliveryMode(mode uint8) client.PublishOption {
	return func(o *client.PublishOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, deliveryMode{}, mode)
	}
}
