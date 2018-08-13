package mock

import (
	"github.com/micro/go-micro/registry"
)

type mockRegistry struct {
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

func (m *mockRegistry) init() {
	// add some mock data
	m.Services = mockData
}

func (m *mockRegistry) GetService(service string) ([]*registry.Service, error) {
	s, ok := m.Services[service]
	if !ok || len(s) == 0 {
		return nil, registry.ErrNotFound
	}
	return s, nil

}

func (m *mockRegistry) ListServices() ([]*registry.Service, error) {
	var services []*registry.Service
	for _, service := range m.Services {
		services = append(services, service...)
	}
	return services, nil
}

func (m *mockRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	services := addServices(m.Services[s.Name], []*registry.Service{s})
	m.Services[s.Name] = services
	return nil
}

func (m *mockRegistry) Deregister(s *registry.Service) error {
	services := delServices(m.Services[s.Name], []*registry.Service{s})
	m.Services[s.Name] = services
	return nil
}

func (m *mockRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	var wopts registry.WatchOptions
	for _, o := range opts {
		o(&wopts)
	}
	return &mockWatcher{exit: make(chan bool), opts: wopts}, nil
}

func (m *mockRegistry) String() string {
	return "mock"
}

func (m *mockRegistry) Init(opts ...registry.Option) error {
	return nil
}

func (m *mockRegistry) Options() registry.Options {
	return registry.Options{}
}

func NewRegistry(opts ...registry.Options) registry.Registry {
	m := &mockRegistry{Services: make(map[string][]*registry.Service)}
	m.init()
	return m
}
