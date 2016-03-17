package mock

import (
	"errors"
	"sync"

	"github.com/micro/go-micro/broker"
	"github.com/pborman/uuid"
)

type MockBroker struct {
	opts broker.Options

	sync.RWMutex
	connected   bool
	Subscribers map[string][]*MockSubscriber
}

type MockPublication struct {
	topic   string
	message *broker.Message
}

type MockSubscriber struct {
	id      string
	topic   string
	exit    chan bool
	handler broker.Handler
	opts    broker.SubscribeOptions
}

func (m *MockBroker) Options() broker.Options {
	return m.opts
}

func (m *MockBroker) Address() string {
	return ""
}

func (m *MockBroker) Connect() error {
	m.Lock()
	defer m.Unlock()

	if m.connected {
		return nil
	}

	m.connected = true

	return nil
}

func (m *MockBroker) Disconnect() error {
	m.Lock()
	defer m.Unlock()

	if !m.connected {
		return nil
	}

	m.connected = false

	return nil
}

func (m *MockBroker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}

func (m *MockBroker) Publish(topic string, message *broker.Message, opts ...broker.PublishOption) error {
	m.Lock()
	defer m.Unlock()

	if !m.connected {
		return errors.New("not connected")
	}

	subs, ok := m.Subscribers[topic]
	if !ok {
		return nil
	}

	p := &MockPublication{
		topic:   topic,
		message: message,
	}

	for _, sub := range subs {
		if err := sub.handler(p); err != nil {
			return err
		}
	}

	return nil
}

func (m *MockBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	m.Lock()
	defer m.Unlock()

	if !m.connected {
		return nil, errors.New("not connected")
	}

	var options broker.SubscribeOptions
	for _, o := range opts {
		o(&options)
	}

	sub := &MockSubscriber{
		exit:    make(chan bool, 1),
		id:      uuid.NewUUID().String(),
		topic:   topic,
		handler: handler,
		opts:    options,
	}

	m.Subscribers[topic] = append(m.Subscribers[topic], sub)

	go func() {
		<-sub.exit
		m.Lock()
		var newSubscribers []*MockSubscriber
		for _, sb := range m.Subscribers[topic] {
			if sb.id == sub.id {
				continue
			}
			newSubscribers = append(newSubscribers, sb)
		}
		m.Subscribers[topic] = newSubscribers
		m.Unlock()
	}()

	return sub, nil
}

func (m *MockBroker) String() string {
	return "mock"
}

func (m *MockPublication) Topic() string {
	return m.topic
}

func (m *MockPublication) Message() *broker.Message {
	return m.message
}

func (m *MockPublication) Ack() error {
	return nil
}

func (m *MockSubscriber) Options() broker.SubscribeOptions {
	return m.opts
}

func (m *MockSubscriber) Topic() string {
	return m.topic
}

func (m *MockSubscriber) Unsubscribe() error {
	m.exit <- true
	return nil
}

func NewBroker(opts ...broker.Option) broker.Broker {
	var options broker.Options
	for _, o := range opts {
		o(&options)
	}

	return &MockBroker{
		opts:        options,
		Subscribers: make(map[string][]*MockSubscriber),
	}
}
