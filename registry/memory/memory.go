// Package memory provides an in-memory registry
package memory

import (
	"sync"

	"github.com/micro/go-micro/registry"
)

type Registry struct {
	sync.RWMutex
	Services map[string][]*registry.Service
}

var (
	// mock data
	Data = map[string][]*registry.Service{
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

// Setup sets mock data
func (m *Registry) Setup() {
	m.Lock()
	defer m.Unlock()

	// add some memory data
	m.Services = Data
}

func (m *Registry) GetService(service string) ([]*registry.Service, error) {
	m.Lock()
	defer m.Unlock()

	s, ok := m.Services[service]
	if !ok || len(s) == 0 {
		return nil, registry.ErrNotFound
	}
	return s, nil

}

func (m *Registry) ListServices() ([]*registry.Service, error) {
	m.Lock()
	defer m.Unlock()

	var services []*registry.Service
	for _, service := range m.Services {
		services = append(services, service...)
	}
	return services, nil
}

func (m *Registry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	m.Lock()
	defer m.Unlock()

	services := addServices(m.Services[s.Name], []*registry.Service{s})
	m.Services[s.Name] = services
	return nil
}

func (m *Registry) Deregister(s *registry.Service) error {
	m.Lock()
	defer m.Unlock()

	services := delServices(m.Services[s.Name], []*registry.Service{s})
	m.Services[s.Name] = services
	return nil
}

func (m *Registry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	var wopts registry.WatchOptions
	for _, o := range opts {
		o(&wopts)
	}
	return &memoryWatcher{exit: make(chan bool), opts: wopts}, nil
}

func (m *Registry) String() string {
	return "memory"
}

func (m *Registry) Init(opts ...registry.Option) error {
	return nil
}

func (m *Registry) Options() registry.Options {
	return registry.Options{}
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	return &Registry{
		Services: make(map[string][]*registry.Service),
	}
}
