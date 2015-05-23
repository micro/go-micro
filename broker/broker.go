package broker

import (
	"code.google.com/p/go-uuid/uuid"
)

type Broker interface {
	Address() string
	Connect() error
	Disconnect() error
	Init() error
	Publish(string, []byte) error
	Subscribe(string, func(*Message)) (Subscriber, error)
}

type Message struct {
	Id        string
	Timestamp int64
	Topic     string
	Data      []byte
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

func Publish(topic string, data []byte) error {
	return DefaultBroker.Publish(topic, data)
}

func Subscribe(topic string, function func(*Message)) (Subscriber, error) {
	return DefaultBroker.Subscribe(topic, function)
}
