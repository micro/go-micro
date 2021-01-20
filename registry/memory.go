package registry

import (
	"context"
	"sync"
	"time"

	"github.com/asim/go-micro/v3/logger"
	"github.com/google/uuid"
)

var (
	sendEventTime = 10 * time.Millisecond
	ttlPruneTime  = time.Second
)

type node struct {
	*Node
	TTL      time.Duration
	LastSeen time.Time
}

type record struct {
	Name      string
	Version   string
	Metadata  map[string]string
	Nodes     map[string]*node
	Endpoints []*Endpoint
}

type memRegistry struct {
	options Options

	sync.RWMutex
	records  map[string]map[string]*record
	watchers map[string]*memWatcher
}

func NewMemoryRegistry(opts ...Option) Registry {
	options := Options{
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}

	records := getServiceRecords(options.Context)
	if records == nil {
		records = make(map[string]map[string]*record)
	}

	reg := &memRegistry{
		options:  options,
		records:  records,
		watchers: make(map[string]*memWatcher),
	}

	go reg.ttlPrune()

	return reg
}

func (m *memRegistry) ttlPrune() {
	prune := time.NewTicker(ttlPruneTime)
	defer prune.Stop()

	for {
		select {
		case <-prune.C:
			m.Lock()
			for name, records := range m.records {
				for version, record := range records {
					for id, n := range record.Nodes {
						if n.TTL != 0 && time.Since(n.LastSeen) > n.TTL {
							if logger.V(logger.DebugLevel, logger.DefaultLogger) {
								logger.Debugf("Registry TTL expired for node %s of service %s", n.Id, name)
							}
							delete(m.records[name][version].Nodes, id)
						}
					}
				}
			}
			m.Unlock()
		}
	}
}

func (m *memRegistry) sendEvent(r *Result) {
	m.RLock()
	watchers := make([]*memWatcher, 0, len(m.watchers))
	for _, w := range m.watchers {
		watchers = append(watchers, w)
	}
	m.RUnlock()

	for _, w := range watchers {
		select {
		case <-w.exit:
			m.Lock()
			delete(m.watchers, w.id)
			m.Unlock()
		default:
			select {
			case w.res <- r:
			case <-time.After(sendEventTime):
			}
		}
	}
}

func (m *memRegistry) Init(opts ...Option) error {
	for _, o := range opts {
		o(&m.options)
	}

	// add services
	m.Lock()
	defer m.Unlock()

	records := getServiceRecords(m.options.Context)
	for name, record := range records {
		// add a whole new service including all of its versions
		if _, ok := m.records[name]; !ok {
			m.records[name] = record
			continue
		}
		// add the versions of the service we dont track yet
		for version, r := range record {
			if _, ok := m.records[name][version]; !ok {
				m.records[name][version] = r
				continue
			}
		}
	}

	return nil
}

func (m *memRegistry) Options() Options {
	return m.options
}

func (m *memRegistry) Register(s *Service, opts ...RegisterOption) error {
	m.Lock()
	defer m.Unlock()

	var options RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	r := serviceToRecord(s, options.TTL)

	if _, ok := m.records[s.Name]; !ok {
		m.records[s.Name] = make(map[string]*record)
	}

	if _, ok := m.records[s.Name][s.Version]; !ok {
		m.records[s.Name][s.Version] = r
		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("Registry added new service: %s, version: %s", s.Name, s.Version)
		}
		go m.sendEvent(&Result{Action: "update", Service: s})
		return nil
	}

	addedNodes := false
	for _, n := range s.Nodes {
		if _, ok := m.records[s.Name][s.Version].Nodes[n.Id]; !ok {
			addedNodes = true
			metadata := make(map[string]string)
			for k, v := range n.Metadata {
				metadata[k] = v
				m.records[s.Name][s.Version].Nodes[n.Id] = &node{
					Node: &Node{
						Id:       n.Id,
						Address:  n.Address,
						Metadata: metadata,
					},
					TTL:      options.TTL,
					LastSeen: time.Now(),
				}
			}
		}
	}

	if addedNodes {
		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("Registry added new node to service: %s, version: %s", s.Name, s.Version)
		}
		go m.sendEvent(&Result{Action: "update", Service: s})
		return nil
	}

	// refresh TTL and timestamp
	for _, n := range s.Nodes {
		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("Updated registration for service: %s, version: %s", s.Name, s.Version)
		}
		m.records[s.Name][s.Version].Nodes[n.Id].TTL = options.TTL
		m.records[s.Name][s.Version].Nodes[n.Id].LastSeen = time.Now()
	}

	return nil
}

func (m *memRegistry) Deregister(s *Service, opts ...DeregisterOption) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.records[s.Name]; ok {
		if _, ok := m.records[s.Name][s.Version]; ok {
			for _, n := range s.Nodes {
				if _, ok := m.records[s.Name][s.Version].Nodes[n.Id]; ok {
					if logger.V(logger.DebugLevel, logger.DefaultLogger) {
						logger.Debugf("Registry removed node from service: %s, version: %s", s.Name, s.Version)
					}
					delete(m.records[s.Name][s.Version].Nodes, n.Id)
				}
			}
			if len(m.records[s.Name][s.Version].Nodes) == 0 {
				delete(m.records[s.Name], s.Version)
				if logger.V(logger.DebugLevel, logger.DefaultLogger) {
					logger.Debugf("Registry removed service: %s, version: %s", s.Name, s.Version)
				}
			}
		}
		if len(m.records[s.Name]) == 0 {
			delete(m.records, s.Name)
			if logger.V(logger.DebugLevel, logger.DefaultLogger) {
				logger.Debugf("Registry removed service: %s", s.Name)
			}
		}
		go m.sendEvent(&Result{Action: "delete", Service: s})
	}

	return nil
}

func (m *memRegistry) GetService(name string, opts ...GetOption) ([]*Service, error) {
	m.RLock()
	defer m.RUnlock()

	records, ok := m.records[name]
	if !ok {
		return nil, ErrNotFound
	}

	services := make([]*Service, len(m.records[name]))
	i := 0
	for _, record := range records {
		services[i] = recordToService(record)
		i++
	}

	return services, nil
}

func (m *memRegistry) ListServices(opts ...ListOption) ([]*Service, error) {
	m.RLock()
	defer m.RUnlock()

	var services []*Service
	for _, records := range m.records {
		for _, record := range records {
			services = append(services, recordToService(record))
		}
	}

	return services, nil
}

func (m *memRegistry) Watch(opts ...WatchOption) (Watcher, error) {
	var wo WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	w := &memWatcher{
		exit: make(chan bool),
		res:  make(chan *Result),
		id:   uuid.New().String(),
		wo:   wo,
	}

	m.Lock()
	m.watchers[w.id] = w
	m.Unlock()

	return w, nil
}

func (m *memRegistry) String() string {
	return "memory"
}
