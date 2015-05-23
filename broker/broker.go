package broker

import (
	"code.google.com/p/go-uuid/uuid"
	"golang.org/x/net/context"
)

type Broker interface {
	Address() string
	Connect() error
	Disconnect() error
	Init() error
	Publish(context.Context, string, []byte) error
	Subscribe(string, func(context.Context, *Message)) (Subscriber, error)
}

type Message struct {
	Id        string
	Timestamp int64
	Topic     string
	Body      []byte
}

type Subscriber interface {
	Topic() string
	Unsubscribe() error
}

type options struct{}

type Option func(*options)

var (
	Address       string
	Id            string
	DefaultBroker Broker
)

func NewBroker(addrs []string, opt ...Option) Broker {
	return newHttpBroker([]string{Address}, opt...)
}

func Init() error {
	if len(Id) == 0 {
		Id = "broker-" + uuid.NewUUID().String()
	}

	if DefaultBroker == nil {
		DefaultBroker = newHttpBroker([]string{Address})
	}

	return DefaultBroker.Init()
}

func Connect() error {
	return DefaultBroker.Connect()
}

func Disconnect() error {
	return DefaultBroker.Disconnect()
}

func Publish(ctx context.Context, topic string, body []byte) error {
	return DefaultBroker.Publish(ctx, topic, body)
}

func Subscribe(topic string, function func(context.Context, *Message)) (Subscriber, error) {
	return DefaultBroker.Subscribe(topic, function)
}
