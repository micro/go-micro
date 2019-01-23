package mock

import (
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/registry"
)

type MockSubscriber struct {
	Id   string
	Opts broker.SubscribeOptions
	Sub  interface{}
}

func (m *MockSubscriber) Topic() string {
	return m.Id
}

func (m *MockSubscriber) Subscriber() interface{} {
	return m.Sub
}

func (m *MockSubscriber) Endpoints() []*registry.Endpoint {
	return []*registry.Endpoint{}
}

func (m *MockSubscriber) Options() broker.SubscribeOptions {
	return m.Opts
}
