package mock

import (
	"github.com/micro/go-micro/registry"
)

type MockRegistry struct{}

func (m *MockRegistry) GetService(service string) ([]*registry.Service, error) {
	return []*registry.Service{
		{
			Name:    "foo",
			Version: "1.0.0",
			Nodes: []*registry.Node{
				{
					Id:      "foo-1.0.0-123",
					Address: "localhost",
					Port:    9999,
				},
				{
					Id:      "foo-1.0.0-321",
					Address: "localhost",
					Port:    9999,
				},
			},
		},
		{
			Name:    "foo",
			Version: "1.0.1",
			Nodes: []*registry.Node{
				{
					Id:      "foo-1.0.1-321",
					Address: "localhost",
					Port:    6666,
				},
			},
		},
		{
			Name:    "foo",
			Version: "1.0.3",
			Nodes: []*registry.Node{
				{
					Id:      "foo-1.0.3-345",
					Address: "localhost",
					Port:    8888,
				},
			},
		},
	}, nil
}

func (m *MockRegistry) ListServices() ([]*registry.Service, error) {
	return []*registry.Service{}, nil
}

func (m *MockRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	return nil
}

func (m *MockRegistry) Deregister(s *registry.Service) error {
	return nil
}

func (m *MockRegistry) Watch() (registry.Watcher, error) {
	return nil, nil
}

func (m *MockRegistry) String() string {
	return "mock"
}

func NewRegistry() *MockRegistry {
	return &MockRegistry{}
}
