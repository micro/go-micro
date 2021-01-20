package stomp

import (
	"context"
	"time"

	"github.com/asim/go-micro/v3/broker"
)

// SubscribeHeaders sets headers for subscriptions
func SubscribeHeaders(h map[string]string) broker.SubscribeOption {
	return setSubscribeOption(subscribeHeaderKey{}, h)
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

// Durable sets a durable subscription
func Durable() broker.SubscribeOption {
	return setSubscribeOption(durableQueueKey{}, true)
}

// Receipt requests a receipt for the delivery should be received
func Receipt(ct time.Duration) broker.PublishOption {
	return setPublishOption(receiptKey{}, true)
}

// SuppressContentLength requests that send does not include a content length
func SuppressContentLength(ct time.Duration) broker.PublishOption {
	return setPublishOption(suppressContentLengthKey{}, true)
}

// ConnectTimeout sets the connection timeout duration
func ConnectTimeout(ct time.Duration) broker.Option {
	return setBrokerOption(connectTimeoutKey{}, ct)
}

// Auth sets the authentication information
func Auth(username string, password string) broker.Option {
	return setBrokerOption(authKey{}, &authRecord{
		username: username,
		password: password,
	})
}

// ConnectHeaders adds headers for the connection
func ConnectHeaders(h map[string]string) broker.Option {
	return setBrokerOption(connectHeaderKey{}, h)
}

// VirtualHost adds host header to define the vhost
func VirtualHost(h string) broker.Option {
	return setBrokerOption(vHostKey{}, h)
}
