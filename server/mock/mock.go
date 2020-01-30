package mock

import (
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v2/server"
)

type MockServer struct {
	sync.Mutex
	Running     bool
	Opts        server.Options
	Handlers    map[string]server.Handler
	Subscribers map[string][]server.Subscriber
}

var (
	_ server.Server = NewServer()
)

func newMockServer(opts ...server.Option) *MockServer {
	var options server.Options

	for _, o := range opts {
		o(&options)
	}

	return &MockServer{
		Opts:        options,
		Handlers:    make(map[string]server.Handler),
		Subscribers: make(map[string][]server.Subscriber),
	}
}

func (m *MockServer) Options() server.Options {
	m.Lock()
	defer m.Unlock()

	return m.Opts
}

func (m *MockServer) Init(opts ...server.Option) error {
	m.Lock()
	defer m.Unlock()

	for _, o := range opts {
		o(&m.Opts)
	}
	return nil
}

func (m *MockServer) Handle(h server.Handler) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.Handlers[h.Name()]; ok {
		return errors.New("Handler " + h.Name() + " already exists")
	}
	m.Handlers[h.Name()] = h
	return nil
}

func (m *MockServer) NewHandler(h interface{}, opts ...server.HandlerOption) server.Handler {
	var options server.HandlerOptions
	for _, o := range opts {
		o(&options)
	}

	return &MockHandler{
		Id:   uuid.New().String(),
		Hdlr: h,
		Opts: options,
	}
}

func (m *MockServer) NewSubscriber(topic string, fn interface{}, opts ...server.SubscriberOption) server.Subscriber {
	var options server.SubscriberOptions
	for _, o := range opts {
		o(&options)
	}

	return &MockSubscriber{
		Id:   topic,
		Sub:  fn,
		Opts: options,
	}
}

func (m *MockServer) Subscribe(sub server.Subscriber) error {
	m.Lock()
	defer m.Unlock()

	subs := m.Subscribers[sub.Topic()]
	subs = append(subs, sub)
	m.Subscribers[sub.Topic()] = subs
	return nil
}

func (m *MockServer) Register() error {
	return nil
}

func (m *MockServer) Deregister() error {
	return nil
}

func (m *MockServer) Start() error {
	m.Lock()
	defer m.Unlock()

	if m.Running {
		return errors.New("already running")
	}

	m.Running = true
	return nil
}

func (m *MockServer) Stop() error {
	m.Lock()
	defer m.Unlock()

	if !m.Running {
		return errors.New("not running")
	}

	m.Running = false
	return nil
}

func (m *MockServer) String() string {
	return "mock"
}

func NewServer(opts ...server.Option) *MockServer {
	return newMockServer(opts...)
}
