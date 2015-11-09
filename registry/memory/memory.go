package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	log "github.com/golang/glog"
	"github.com/hashicorp/memberlist"
	"github.com/piemapping/go-micro/registry"
	"github.com/pborman/uuid"
)

type action int

const (
	addAction action = iota
	delAction
	syncAction
)

type broadcast struct {
	msg    []byte
	notify chan<- struct{}
}

type delegate struct {
	broadcasts *memberlist.TransmitLimitedQueue
	updates    chan *update
}

type memoryRegistry struct {
	broadcasts *memberlist.TransmitLimitedQueue
	updates    chan *update

	sync.RWMutex
	services map[string]*registry.Service
}

type update struct {
	Action  action
	Service *registry.Service
	sync    chan *registry.Service
}

type watcher struct{}

func (b *broadcast) Invalidates(other memberlist.Broadcast) bool {
	return false
}

func (b *broadcast) Message() []byte {
	return b.msg
}

func (b *broadcast) Finished() {
	if b.notify != nil {
		close(b.notify)
	}
}

func (d *delegate) NodeMeta(limit int) []byte {
	return []byte{}
}

func (d *delegate) NotifyMsg(b []byte) {
	if len(b) == 0 {
		return
	}

	buf := make([]byte, len(b))
	copy(buf, b)

	go func() {
		switch buf[0] {
		case 'd': // data
			var updates []*update
			if err := json.Unmarshal(buf[1:], &updates); err != nil {
				return
			}
			for _, u := range updates {
				d.updates <- u
			}
		}
	}()
}

func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.broadcasts.GetBroadcasts(overhead, limit)
}

func (d *delegate) LocalState(join bool) []byte {
	if !join {
		return []byte{}
	}

	syncCh := make(chan *registry.Service, 1)
	m := map[string]*registry.Service{}

	d.updates <- &update{
		Action: syncAction,
		sync:   syncCh,
	}

	for s := range syncCh {
		m[s.Name] = s
	}

	b, _ := json.Marshal(m)
	return b
}

func (d *delegate) MergeRemoteState(buf []byte, join bool) {
	if len(buf) == 0 {
		return
	}
	if !join {
		return
	}

	var m map[string]*registry.Service
	if err := json.Unmarshal(buf, &m); err != nil {
		return
	}

	for _, service := range m {
		d.updates <- &update{
			Action:  addAction,
			Service: service,
			sync:    nil,
		}
	}
}

func (m *memoryRegistry) run() {
	for u := range m.updates {
		switch u.Action {
		case addAction:
			m.Lock()
			m.services[u.Service.Name] = u.Service
			m.Unlock()
		case delAction:
			m.Lock()
			delete(m.services, u.Service.Name)
			m.Unlock()
		case syncAction:
			if u.sync == nil {
				continue
			}
			m.RLock()
			for _, service := range m.services {
				u.sync <- service
			}
			m.RUnlock()
			close(u.sync)
		}
	}
}

func (m *memoryRegistry) Register(s *registry.Service) error {
	m.Lock()
	m.services[s.Name] = s
	m.Unlock()

	b, _ := json.Marshal([]*update{
		&update{
			Action:  addAction,
			Service: s,
		},
	})

	m.broadcasts.QueueBroadcast(&broadcast{
		msg:    append([]byte("d"), b...),
		notify: nil,
	})

	return nil
}

func (m *memoryRegistry) Deregister(s *registry.Service) error {
	m.Lock()
	delete(m.services, s.Name)
	m.Unlock()

	b, _ := json.Marshal([]*update{
		&update{
			Action:  delAction,
			Service: s,
		},
	})

	m.broadcasts.QueueBroadcast(&broadcast{
		msg:    append([]byte("d"), b...),
		notify: nil,
	})

	return nil
}

func (m *memoryRegistry) GetService(name string) (*registry.Service, error) {
	m.RLock()
	service, ok := m.services[name]
	m.RUnlock()
	if !ok {
		return nil, fmt.Errorf("Service %s not found", name)
	}
	return service, nil
}

func (m *memoryRegistry) ListServices() ([]*registry.Service, error) {
	var services []*registry.Service
	m.RLock()
	for _, service := range m.services {
		services = append(services, service)
	}
	m.RUnlock()
	return services, nil
}

func (m *memoryRegistry) Watch() (registry.Watcher, error) {
	return &watcher{}, nil
}

func (w *watcher) Stop() {
	return
}

func NewRegistry(addrs []string, opt ...registry.Option) registry.Registry {
	cAddrs := []string{}
	hostname, _ := os.Hostname()
	updates := make(chan *update, 100)

	for _, addr := range addrs {
		if len(addr) > 0 {
			cAddrs = append(cAddrs, addr)
		}
	}

	broadcasts := &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return len(cAddrs)
		},
		RetransmitMult: 3,
	}

	mr := &memoryRegistry{
		broadcasts: broadcasts,
		services:   make(map[string]*registry.Service),
		updates:    updates,
	}

	go mr.run()

	c := memberlist.DefaultLocalConfig()
	c.BindPort = 0
	c.Name = hostname + "-" + uuid.NewUUID().String()
	c.Delegate = &delegate{
		updates:    updates,
		broadcasts: broadcasts,
	}

	m, err := memberlist.Create(c)
	if err != nil {
		log.Fatalf("Error creating memberlist: %v", err)
	}

	if len(cAddrs) > 0 {
		_, err := m.Join(cAddrs)
		if err != nil {
			log.Fatalf("Error joining members: %v", err)
		}
	}

	log.Infof("Local memberlist node %s:%d\n", m.LocalNode().Addr, m.LocalNode().Port)
	return mr
}
