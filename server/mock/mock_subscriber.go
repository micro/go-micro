package mock

import (
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/server"
)

type MockSubscriber struct {
	Id   string
	Opts server.SubscriberOptions
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

func (m *MockSubscriber) Options() server.SubscriberOptions {
	return m.Opts
}
