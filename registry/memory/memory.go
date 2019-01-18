// Package memory provides an in-memory registry
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/registry"
)

type Registry struct {
	options registry.Options

	sync.RWMutex
	Services map[string][]*registry.Service
	Watchers map[string]*Watcher
}

var (
	timeout = time.Millisecond * 10
)

// Setup sets mock data
func (m *Registry) Setup() {
	m.Lock()
	defer m.Unlock()

	// add some memory data
	m.Services = Data
}

func (m *Registry) watch(r *registry.Result) {
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
		m.Services[k] = addServices(s, v)
	}
	m.Unlock()
	return nil
}

func (m *Registry) Options() registry.Options {
	return m.options
}

func (m *Registry) GetService(service string) ([]*registry.Service, error) {
	m.RLock()
	s, ok := m.Services[service]
	if !ok || len(s) == 0 {
		m.RUnlock()
		return nil, registry.ErrNotFound
	}
	m.RUnlock()
	return s, nil
}

func (m *Registry) ListServices() ([]*registry.Service, error) {
	m.RLock()
	var services []*registry.Service
	for _, service := range m.Services {
		services = append(services, service...)
	}
	m.RUnlock()
	return services, nil
}

func (m *Registry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	go m.watch(&registry.Result{Action: "update", Service: s})

	m.Lock()
	services := addServices(m.Services[s.Name], []*registry.Service{s})
	m.Services[s.Name] = services
	m.Unlock()
	return nil
}

func (m *Registry) Deregister(s *registry.Service) error {
	go m.watch(&registry.Result{Action: "delete", Service: s})

	m.Lock()
	services := delServices(m.Services[s.Name], []*registry.Service{s})
	m.Services[s.Name] = services
	m.Unlock()
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
