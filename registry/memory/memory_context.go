package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/util/log"
)

type contextNode struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type ContextRegistry struct {
	options registry.Options

	sync.RWMutex
	Services map[string][]*registry.Service
	nodes    map[string]map[string]contextNode
	nodeLock sync.Mutex
	Watchers map[string]*Watcher
}

func NewContextRegistry(opts ...registry.Option) registry.Registry {
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

	reg := &ContextRegistry{
		options:  options,
		Services: services,
		nodes:    make(map[string]map[string]contextNode),
		Watchers: make(map[string]*Watcher),
	}

	return reg
}

func (m *ContextRegistry) sendEvent(r *registry.Result) {
	watchers := make([]*Watcher, 0, len(m.Watchers))

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
			case <-time.After(sendEventTime):
			}
		}
	}
}

func (m *ContextRegistry) Init(opts ...registry.Option) error {
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

func (m *ContextRegistry) Options() registry.Options {
	return m.options
}

func (m *ContextRegistry) cleanServices() {
	m.nodeLock.Lock()
	defer m.nodeLock.Unlock()
	for name, service := range m.nodes {
		for nodeId, node := range service {
			select {
			case <-node.ctx.Done():
				m.Lock()
				for _, s := range m.Services[name] {
					list := []int{}
					for x, node := range s.Nodes {
						if nodeTrackId(name, s.Version, node.Id) == nodeId {
							list = append(list, x)
						}
					}

					for _, x := range list {
						if x > len(s.Nodes)-1 {
							x = len(s.Nodes) - 1
						}

						s.Nodes = append(s.Nodes[:x], s.Nodes[x+1:]...)
					}
				}
				m.Unlock()
			default:
			}
		}
	}
}

func (m *ContextRegistry) GetService(name string) ([]*registry.Service, error) {
	m.cleanServices()
	m.RLock()
	service, ok := m.Services[name]
	m.RUnlock()
	if !ok {
		return nil, registry.ErrNotFound
	}

	return service, nil
}

func (m *ContextRegistry) ListServices() ([]*registry.Service, error) {
	m.cleanServices()
	var services []*registry.Service
	m.RLock()
	for _, service := range m.Services {
		services = append(services, service...)
	}
	m.RUnlock()
	return services, nil
}

func (m *ContextRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	m.cleanServices()
	m.Lock()
	defer m.Unlock()

	log.Debugf("[memory] Registry registering service: %s", s.Name)

	var options registry.RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	if service, ok := m.Services[s.Name]; !ok {
		m.Services[s.Name] = []*registry.Service{s}
		// add all nodes into nodes map to track their TTL
		for _, n := range s.Nodes {
			log.Debugf("[memory] Registry tracking new service: %s, node %s", s.Name, n.Id)
			ctx, cancel := context.WithTimeout(context.Background(), options.TTL)
			m.nodeLock.Lock()
			if _, ok := m.nodes[s.Name]; !ok {
				m.nodes[s.Name] = map[string]contextNode{}
			}
			m.nodes[s.Name][nodeTrackId(s.Name, s.Version, n.Id)] = contextNode{ctx, cancel}
			m.nodeLock.Unlock()
		}
		go m.sendEvent(&registry.Result{Action: "update", Service: s})
		return nil
	} else {
		// svcCount keeps the count of all versions of particular service
		//svcCount := len(service)
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
		// if the node count of any service [version] changed we know we are adding a new node to the service
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
						log.Debugf("[memory] Registry tracking new node: %s for service %s", n.Id, s.Name)
						m.nodeLock.Lock()
						ctx, cancel := context.WithTimeout(context.Background(), options.TTL)
						if _, ok := m.nodes[s.Name]; !ok {
							m.nodes[s.Name] = map[string]contextNode{}
						}
						m.nodes[s.Name][nodeTrackId(s.Name, s.Version, n.Id)] = contextNode{ctx, cancel}
						m.nodeLock.Unlock()
					}
				}
				m.Services[s.Name] = merged
				go m.sendEvent(&registry.Result{Action: "update", Service: s})
				return nil
			}
			// refresh the timestamp and TTL of the service node
			for _, n := range s.Nodes {
				trackId := nodeTrackId(s.Name, s.Version, n.Id)
				log.Debugf("[memory] Registry refreshing TTL for node %s for service %s", n.Id, s.Name)
				m.nodeLock.Lock()
				if _, ok := m.nodes[s.Name]; !ok {
					m.nodes[s.Name] = map[string]contextNode{}
				}
				if trackedNode, ok := m.nodes[s.Name][trackId]; ok {
					select {
					case <-trackedNode.ctx.Done():
					default:
						ctx, cancel := context.WithTimeout(context.Background(), options.TTL)
						m.nodes[s.Name][trackId] = contextNode{ctx, cancel}
					}
				}
				m.nodeLock.Unlock()
			}
		}
	}

	return nil
}

func (m *ContextRegistry) Deregister(s *registry.Service) error {
	m.cleanServices()
	m.Lock()
	defer m.Unlock()

	log.Debugf("[memory] Registry deregistering service: %s", s.Name)

	if service, ok := m.Services[s.Name]; ok {
		// svcNodes collects the list of all node Ids for each service version
		svcNodes := make(map[string]map[string][]string)
		// collect all service node ids for all service versions
		for _, svc := range service {
			if _, ok := svcNodes[svc.Name]; !ok {
				svcNodes[svc.Name] = make(map[string][]string)
			}
			if _, ok := svcNodes[svc.Name][svc.Version]; !ok {
				for _, n := range svc.Nodes {
					svcNodes[svc.Name][svc.Version] = append(svcNodes[svc.Name][svc.Version], n.Id)
				}
			}
		}
		// if there are no more services we know we have either removed all nodes or there were no nodes
		if updatedService := registry.Remove(service, []*registry.Service{s}); len(updatedService) == 0 {
			for _, id := range svcNodes[s.Name][s.Version] {
				log.Debugf("[memory] Registry stopped tracking node %s for service %s", id, s.Name)
				id := nodeTrackId(s.Name, s.Version, id)
				if _, ok := m.nodes[s.Name]; ok && m.nodes[s.Name][id].cancel != nil {
					m.nodes[s.Name][id].cancel()
				}
				delete(m.nodes, id)
				go m.sendEvent(&registry.Result{Action: "delete", Service: s})
			}
			log.Debugf("[memory] Registry deleting service %s: no service nodes", s.Name)
			delete(m.Services, s.Name)
			return nil
		} else {
			// find out which nodes have been removed
			for _, id := range svcNodes[s.Name][s.Version] {
				for _, svc := range updatedService {
					var found bool
					for _, n := range svc.Nodes {
						if id == n.Id {
							found = true
							break
						}
					}
					if !found {
						log.Debugf("[memory] Registry stopped tracking node %s for service %s", id, s.Name)
						id := nodeTrackId(s.Name, s.Version, id)
						if _, ok := m.nodes[s.Name]; ok && m.nodes[s.Name][id].cancel != nil {
							m.nodes[s.Name][id].cancel()
						}
						delete(m.nodes, id)
						go m.sendEvent(&registry.Result{Action: "delete", Service: s})
					}
				}
				m.Services[s.Name] = updatedService
			}
		}
	}

	return nil
}

func (m *ContextRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
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

func (m *ContextRegistry) String() string {
	return "memory_context"
}
