// Package memory provides an in-memory registry
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/util/log"
)

var (
	timeout = time.Millisecond * 10
)

// node tracks node registration timestamp and TTL
type node struct {
	ts  time.Time
	ttl time.Duration
}

type Registry struct {
	options registry.Options

	sync.RWMutex
	Services map[string][]*registry.Service
	nodes    map[string]*node
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
		nodes:    make(map[string]*node),
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

	var options registry.RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	if service, ok := m.Services[s.Name]; !ok {
		m.Services[s.Name] = []*registry.Service{s}
		// add all nodes into nodes map to track their TTL
		for _, n := range s.Nodes {
			log.Logf("Tracking node %s for service %s", n.Id, s.Name)
			m.nodes[s.Name+n.Id] = &node{
				ts:  time.Now(),
				ttl: options.TTL,
			}
		}
		go m.sendEvent(&registry.Result{Action: "update", Service: s})
	} else {
		// svcCount essentially keep the count of all service vesions
		svcCount := len(service)
		// svcNodes maintains a list of node Ids per particular service version
		svcNodes := make(map[string]map[string][]string)
		// collect all service ids for all service versions
		for _, s := range service {
			if _, ok := svcNodes[s.Name]; !ok {
				svcNodes[s.Name] = make(map[string][]string)
			}
			if _, ok := svcNodes[s.Name][s.Version]; !ok {
				for _, n := range s.Nodes {
					svcNodes[s.Name][s.Version] = append(svcNodes[s.Name][s.Version], n.Id)
				}
			}
		}
		// if merged count and original service counts changed we know we are adding a new version of the service
		merged := registry.Merge(service, []*registry.Service{s})
		if len(merged) != svcCount {
			m.Services[s.Name] = merged
			// we know s is the new [version of] service; we need to strart tracking its nodes
			for _, n := range s.Nodes {
				log.Logf("Tracking node %s for service %s", n.Id, s.Name)
				m.nodes[s.Name+n.Id] = &node{
					ts:  time.Now(),
					ttl: options.TTL,
				}
			}
			go m.sendEvent(&registry.Result{Action: "update", Service: s})
			return nil
		}
		// if the node count of any particular service [version] changed we know we are adding a new node to the service
		for _, s := range merged {
			// we know that if the node counts have changed we need to track new nodes
			if len(s.Nodes) != len(svcNodes[s.Name][s.Version]) {
				for _, n := range s.Nodes {
					var found bool
					for _, id := range svcNodes[s.Name][s.Version] {
						if n.Id == id {
							found = true
							break
						}
					}
					if !found {
						log.Logf("Tracking node %s for service %s", n.Id, s.Name)
						m.nodes[s.Name+n.Id] = &node{
							ts:  time.Now(),
							ttl: options.TTL,
						}
					}
				}
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
		// svcNodes maintains a list of node Ids per particular service version
		svcNodes := make(map[string]map[string][]string)
		// collect all service ids for all service versions
		for _, s := range service {
			if _, ok := svcNodes[s.Name]; !ok {
				svcNodes[s.Name] = make(map[string][]string)
			}
			if _, ok := svcNodes[s.Name][s.Version]; !ok {
				for _, n := range s.Nodes {
					svcNodes[s.Name][s.Version] = append(svcNodes[s.Name][s.Version], n.Id)
				}
			}
		}
		if service := registry.Remove(service, []*registry.Service{s}); len(service) == 0 {
			id := svcNodes[s.Name][s.Version][0]
			log.Logf("Stopped tracking node %s for service %s", id, s.Name)
			delete(m.nodes, s.Name+id)
			delete(m.Services, s.Name)
		} else {
			// find out which nodes have been removed
			for _, id := range svcNodes[s.Name][s.Version] {
				for _, s := range service {
					var found bool
					for _, n := range s.Nodes {
						if id == n.Id {
							found = true
							break
						}
					}
					if !found {
						log.Logf("Stopped tracking node %s for service %s", id, s.Name)
						delete(m.nodes, s.Name+id)
					}
				}
				m.Services[s.Name] = service
			}
		}
		go m.sendEvent(&registry.Result{Action: "delete", Service: s})
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
