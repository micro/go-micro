// Package memory provides an in-memory registry
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/registry"
)

var (
	timeout = time.Millisecond * 10
)

type Registry struct {
	options registry.Options

	sync.RWMutex
	Services map[string][]*registry.Service
	Watchers map[string]*Watcher
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	options := registry.Options{
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}

	services := getServices(options.Context)
	if services == nil {
		services = make(map[string][]*registry.Service)
	}

	return &Registry{
		options:  options,
		Services: services,
		Watchers: make(map[string]*Watcher),
	}
}

func (m *Registry) sendEvent(r *registry.Result) {
	var watchers []*Watcher

	m.RLock()
	for _, w := range m.Watchers {
		watchers = append(watchers, w)
	}
	m.RUnlock()

	for _, w := range watchers {
		select {
		case <-w.exit:
			m.Lock()
			delete(m.Watchers, w.id)
			m.Unlock()
		default:
			select {
			case w.res <- r:
			case <-time.After(timeout):
			}
		}
	}
}

func (m *Registry) Init(opts ...registry.Option) error {
	for _, o := range opts {
		o(&m.options)
	}

	// add services
	m.Lock()
	for k, v := range getServices(m.options.Context) {
		s := m.Services[k]
		m.Services[k] = registry.Merge(s, v)
	}
	m.Unlock()
	return nil
}

func (m *Registry) Options() registry.Options {
	return m.options
}

func (m *Registry) GetService(name string) ([]*registry.Service, error) {
	m.RLock()
	service, ok := m.Services[name]
	m.RUnlock()
	if !ok {
		return nil, registry.ErrNotFound
	}

	return service, nil
}

func (m *Registry) ListServices() ([]*registry.Service, error) {
	var services []*registry.Service
	m.RLock()
	for _, service := range m.Services {
		services = append(services, service...)
	}
	m.RUnlock()
	return services, nil
}

func (m *Registry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	m.Lock()
	defer m.Unlock()

	if service, ok := m.Services[s.Name]; !ok {
		m.Services[s.Name] = []*registry.Service{s}
		go m.sendEvent(&registry.Result{Action: "update", Service: s})
	} else {
		svcCount := len(service)
		svcNodeCounts := make(map[string]map[string]int)
		for _, s := range service {
			if _, ok := svcNodeCounts[s.Name]; !ok {
				svcNodeCounts[s.Name] = make(map[string]int)
			}
			if _, ok := svcNodeCounts[s.Name][s.Version]; !ok {
				svcNodeCounts[s.Name][s.Version] = len(s.Nodes)
			}
		}
		// if merged count and original service counts changed we added new version of the service
		merged := registry.Merge(service, []*registry.Service{s})
		if len(merged) != svcCount {
			m.Services[s.Name] = merged
			go m.sendEvent(&registry.Result{Action: "update", Service: s})
			return nil
		}
		// if the node count for a particular service has changed we added a new node to the service
		for _, s := range merged {
			if len(s.Nodes) != svcNodeCounts[s.Name][s.Version] {
				m.Services[s.Name] = merged
				go m.sendEvent(&registry.Result{Action: "update", Service: s})
				return nil
			}
		}
	}

	return nil
}

func (m *Registry) Deregister(s *registry.Service) error {
	m.Lock()
	defer m.Unlock()

	if service, ok := m.Services[s.Name]; ok {
		go m.sendEvent(&registry.Result{Action: "delete", Service: s})
		if service := registry.Remove(service, []*registry.Service{s}); len(service) == 0 {
			delete(m.Services, s.Name)
		} else {
			m.Services[s.Name] = service
		}
	}

	return nil
}

func (m *Registry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	w := &Watcher{
		exit: make(chan bool),
		res:  make(chan *registry.Result),
		id:   uuid.New().String(),
		wo:   wo,
	}

	m.Lock()
	m.Watchers[w.id] = w
	m.Unlock()
	return w, nil
}

func (m *Registry) String() string {
	return "memory"
}
