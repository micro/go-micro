package mock

import (
	"github.com/micro/go-micro/registry"
)

type MockRegistry struct {
	Services map[string][]*registry.Service
}

var (
	mockData = map[string][]*registry.Service{
		"foo": []*registry.Service{
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
		},
	}
)

func (m *MockRegistry) init() {
	// add some mock data
	m.Services = mockData
}

func (m *MockRegistry) GetService(service string) ([]*registry.Service, error) {
	s, ok := m.Services[service]
	if !ok {
		return nil, registry.ErrNotFound
	}
	return s, nil

}

func (m *MockRegistry) ListServices() ([]*registry.Service, error) {
	var services []*registry.Service
	for _, service := range m.Services {
		services = append(services, service...)
	}
	return services, nil
}

func (m *MockRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	services := addServices(m.Services[s.Name], []*registry.Service{s})
	m.Services[s.Name] = services
	return nil
}

func (m *MockRegistry) Deregister(s *registry.Service) error {
	services := delServices(m.Services[s.Name], []*registry.Service{s})
	m.Services[s.Name] = services
	return nil
}

func (m *MockRegistry) Watch() (registry.Watcher, error) {
	return nil, nil
}

func (m *MockRegistry) String() string {
	return "mock"
}

func NewRegistry() *MockRegistry {
	m := &MockRegistry{Services: make(map[string][]*registry.Service)}
	m.init()
	return m
}
