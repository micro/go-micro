// Package memory provides an in-memory registry
package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/util/log"
)

var (
	sendEventTime = 10 * time.Millisecond
	ttlPruneTime  = 1 * time.Minute
	DefaultTTL    = 1 * time.Minute
)

// node tracks node registration timestamp and TTL
type node struct {
	lastSeen time.Time
	ttl      time.Duration
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

	reg := &Registry{
		options:  options,
		Services: services,
		nodes:    make(map[string]*node),
		Watchers: make(map[string]*Watcher),
	}

	go reg.ttlPrune()

	return reg
}

// nodeTrackId returns a string we use to track a node of a given service
func nodeTrackId(svcName, svcVersion, nodeId string) string {
	return svcName + "+" + svcVersion + "+" + nodeId
}

func (m *Registry) ttlPrune() {
	prune := time.NewTicker(ttlPruneTime)
	defer prune.Stop()

	for {
		select {
		case <-prune.C:
			m.Lock()
			for nodeTrackId, node := range m.nodes {
				// if the TTL has been set and we exceed the hresholdset by it we stop tracking the node
				if node.ttl.Seconds() != 0.0 && time.Since(node.lastSeen) > node.ttl {
					// split nodeTrackID into service Name, Version and Node Id
					trackIdSplit := strings.Split(nodeTrackId, "+")
					svcName, svcVersion, nodeId := trackIdSplit[0], trackIdSplit[1], trackIdSplit[2]
					log.Debugf("Registry TTL expired for service %s, node %s", svcName, nodeId)
					// we need to find a node that expired and delete it from service nodes
					if _, ok := m.Services[svcName]; ok {
						for _, service := range m.Services[svcName] {
							if service.Version != svcVersion {
								continue
							}
							// find expired service node and delete it
							var nodes []*registry.Node
							for _, n := range service.Nodes {
								var del bool
								if n.Id == nodeId {
									del = true
								}
								if !del {
									nodes = append(nodes, n)
								}
							}
							service.Nodes = nodes
						}
					}
					// stop tracking the node
					delete(m.nodes, nodeTrackId)
				}
			}
			m.Unlock()
		}
	}

	return
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
			case <-time.After(sendEventTime):
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

	log.Debugf("Registry deregistering service: %s", s.Name)

	var options registry.RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	if service, ok := m.Services[s.Name]; !ok {
		m.Services[s.Name] = []*registry.Service{s}
		// add all nodes into nodes map to track their TTL
		for _, n := range s.Nodes {
			log.Debugf("Registry tracking new service: %s, node %s", s.Name, n.Id)
			m.nodes[nodeTrackId(s.Name, s.Version, n.Id)] = &node{
				lastSeen: time.Now(),
				ttl:      options.TTL,
			}
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
						log.Debugf("Registry tracking new node: %s for service %s", n.Id, s.Name)
						m.nodes[nodeTrackId(s.Name, s.Version, n.Id)] = &node{
							lastSeen: time.Now(),
							ttl:      options.TTL,
						}
					}
				}
				m.Services[s.Name] = merged
				go m.sendEvent(&registry.Result{Action: "update", Service: s})
				return nil
			}
			// refresh the timestamp and TTL of the service node
			for _, n := range s.Nodes {
				trackId := nodeTrackId(s.Name, s.Version, n.Id)
				log.Debugf("Registry refreshing TTL for node %s for service %s", n.Id, s.Name)
				if trackedNode, ok := m.nodes[trackId]; ok {
					trackedNode.lastSeen = time.Now()
					trackedNode.ttl = options.TTL
				}
			}
		}
	}

	return nil
}

func (m *Registry) Deregister(s *registry.Service) error {
	m.Lock()
	defer m.Unlock()

	log.Debugf("Registry deregistering service: %s", s.Name)

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
				log.Debugf("Registry stopped tracking node %s for service %s", id, s.Name)
				delete(m.nodes, nodeTrackId(s.Name, s.Version, id))
				go m.sendEvent(&registry.Result{Action: "delete", Service: s})
			}
			log.Debugf("Registry deleting service %s: no service nodes", s.Name)
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
						log.Debugf("Registry stopped tracking node %s for service %s", id, s.Name)
						delete(m.nodes, nodeTrackId(s.Name, s.Version, id))
						go m.sendEvent(&registry.Result{Action: "delete", Service: s})
					}
				}
				m.Services[s.Name] = updatedService
			}
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
