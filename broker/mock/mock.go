package mock

import (
	"errors"
	"sync"

	"github.com/micro/go-micro/broker"
	"github.com/pborman/uuid"
)

type mockBroker struct {
	opts broker.Options

	sync.RWMutex
	connected   bool
	Subscribers map[string][]*mockSubscriber
}

type mockPublication struct {
	topic   string
	message *broker.Message
}

type mockSubscriber struct {
	id      string
	topic   string
	exit    chan bool
	handler broker.Handler
	opts    broker.SubscribeOptions
}

func (m *mockBroker) Options() broker.Options {
	return m.opts
}

func (m *mockBroker) Address() string {
	return ""
}

func (m *mockBroker) Connect() error {
	m.Lock()
	defer m.Unlock()

	if m.connected {
		return nil
	}

	m.connected = true

	return nil
}

func (m *mockBroker) Disconnect() error {
	m.Lock()
	defer m.Unlock()

	if !m.connected {
		return nil
	}

	m.connected = false

	return nil
}

func (m *mockBroker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}

func (m *mockBroker) Publish(topic string, message *broker.Message, opts ...broker.PublishOption) error {
	m.Lock()
	defer m.Unlock()

	if !m.connected {
		return errors.New("not connected")
	}

	subs, ok := m.Subscribers[topic]
	if !ok {
		return nil
	}

	p := &mockPublication{
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

func (m *mockBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	m.Lock()
	defer m.Unlock()

	if !m.connected {
		return nil, errors.New("not connected")
	}

	var options broker.SubscribeOptions
	for _, o := range opts {
		o(&options)
	}

	sub := &mockSubscriber{
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
		var newSubscribers []*mockSubscriber
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

func (m *mockBroker) String() string {
	return "mock"
}

func (m *mockPublication) Topic() string {
	return m.topic
}

func (m *mockPublication) Message() *broker.Message {
	return m.message
}

func (m *mockPublication) Ack() error {
	return nil
}

func (m *mockSubscriber) Options() broker.SubscribeOptions {
	return m.opts
}

func (m *mockSubscriber) Topic() string {
	return m.topic
}

func (m *mockSubscriber) Unsubscribe() error {
	m.exit <- true
	return nil
}

func NewBroker(opts ...broker.Option) broker.Broker {
	var options broker.Options
	for _, o := range opts {
		o(&options)
	}

	return &mockBroker{
		opts:        options,
		Subscribers: make(map[string][]*mockSubscriber),
	}
}
