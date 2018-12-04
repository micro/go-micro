// Package gossip provides a zero dependency registry using the gossip protocol SWIM
package gossip

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/hashicorp/memberlist"
	"github.com/micro/go-log"
	"github.com/micro/go-micro/registry"
	pb "github.com/micro/go-micro/registry/gossip/proto"
)

type gossipRegistry struct {
	opts       registry.Options
	queue      *memberlist.TransmitLimitedQueue
	memberlist *memberlist.Memberlist
	delegate   *delegate

	sync.RWMutex
	services map[string][]*registry.Service
	watchers map[string]*watcher
}

var (
	defaultPort = 8118
)

type broadcast struct {
	update *pb.Update
	notify chan<- struct{}
}

type delegate struct {
	queue    *memberlist.TransmitLimitedQueue
	registry *gossipRegistry
}

func (b *broadcast) Invalidates(other memberlist.Broadcast) bool {
	up := new(pb.Update)
	if err := proto.Unmarshal(other.Message(), up); err != nil {
		return false
	}

	// ids do not match
	if b.update.Id == up.Id {
		return false
	}

	// timestamps do not match
	if b.update.Timestamp != up.Timestamp {
		return false
	}

	// type does not match
	if b.update.Type != up.Type {
		return false
	}

	// invalidates
	return true
}

func (b *broadcast) Message() []byte {
	up, err := proto.Marshal(b.update)
	if err != nil {
		return nil
	}
	return up
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

	up := new(pb.Update)
	if err := proto.Unmarshal(b, up); err != nil {
		return
	}

	// only process service action
	if up.Type != "service" {
		return
	}

	var service *registry.Service

	switch up.Metadata["Content-Type"] {
	case "application/json":
		if err := json.Unmarshal(up.Data, &service); err != nil {
			return
		}
	// no other content type
	default:
		return
	}

	d.registry.Lock()
	defer d.registry.Unlock()

	// get existing service
	s := d.registry.services[service.Name]

	// save update
	switch up.Action {
	case "update":
		d.registry.services[service.Name] = addServices(s, []*registry.Service{service})
	case "delete":
		services := delServices(s, []*registry.Service{service})
		if len(services) == 0 {
			delete(d.registry.services, service.Name)
			return
		}
		d.registry.services[service.Name] = services
	default:
		return
	}

	// notify watchers
	for _, w := range d.registry.watchers {
		select {
		case w.ch <- &registry.Result{Action: up.Action, Service: service}:
		default:
		}
	}
}

func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.queue.GetBroadcasts(overhead, limit)
}

func (d *delegate) LocalState(join bool) []byte {
	d.registry.RLock()
	b, _ := json.Marshal(d.registry.services)
	d.registry.RUnlock()
	return b
}

func (d *delegate) MergeRemoteState(buf []byte, join bool) {
	if len(buf) == 0 {
		return
	}
	if !join {
		return
	}
	var services map[string][]*registry.Service
	if err := json.Unmarshal(buf, &services); err != nil {
		return
	}

	d.registry.Lock()
	for k, v := range services {
		d.registry.services[k] = addServices(d.registry.services[k], v)
	}
	d.registry.Unlock()
}

func (g *gossipRegistry) Init(opts ...registry.Option) error {
	addrs := g.opts.Addrs
	for _, o := range opts {
		o(&g.opts)
	}

	// if we have memberlist join it
	if len(addrs) != len(g.opts.Addrs) {
		_, err := g.memberlist.Join(g.opts.Addrs)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *gossipRegistry) Options() registry.Options {
	return g.opts
}

func (g *gossipRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}

	g.Lock()
	g.services[s.Name] = addServices(g.services[s.Name], []*registry.Service{s})
	g.Unlock()

	up := &pb.Update{
		Id:        uuid.New().String(),
		Timestamp: uint64(time.Now().UnixNano()),
		Action:    "update",
		Type:      "service",
		Metadata: map[string]string{
			"Content-Type": "application/json",
		},
		Data: b,
	}

	g.queue.QueueBroadcast(&broadcast{
		update: up,
		notify: nil,
	})

	return nil
}

func (g *gossipRegistry) Deregister(s *registry.Service) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}

	g.Lock()
	g.services[s.Name] = delServices(g.services[s.Name], []*registry.Service{s})
	g.Unlock()

	up := &pb.Update{
		Id:        uuid.New().String(),
		Timestamp: uint64(time.Now().UnixNano()),
		Action:    "delete",
		Type:      "service",
		Metadata: map[string]string{
			"Content-Type": "application/json",
		},
		Data: b,
	}

	g.queue.QueueBroadcast(&broadcast{
		update: up,
		notify: nil,
	})

	return nil
}

func (g *gossipRegistry) GetService(name string) ([]*registry.Service, error) {
	g.RLock()
	if s, ok := g.services[name]; ok {
		service := cp(s)
		g.RUnlock()
		return service, nil
	}
	g.RUnlock()
	return nil, registry.ErrNotFound
}

func (g *gossipRegistry) ListServices() ([]*registry.Service, error) {
	var services []*registry.Service
	g.RLock()
	for name, _ := range g.services {
		services = append(services, &registry.Service{Name: name})
	}
	g.RUnlock()
	return services, nil
}

func (g *gossipRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	var options registry.WatchOptions
	for _, o := range opts {
		o(&options)
	}

	// watcher id
	id := uuid.New().String()

	// create watcher
	w := &watcher{
		ch:   make(chan *registry.Result, 1),
		exit: make(chan bool),
		id:   id,
		// filter service
		srv: options.Service,
		// delete self
		fn: func() {
			g.Lock()
			delete(g.watchers, id)
			g.Unlock()
		},
	}

	// save watcher
	g.Lock()
	g.watchers[w.id] = w
	g.Unlock()

	return w, nil
}

func (g *gossipRegistry) String() string {
	return "gossip"
}

func (g *gossipRegistry) run() error {
	hostname, _ := os.Hostname()

	// delegates
	d := new(delegate)

	// create a new default config
	c := memberlist.DefaultLocalConfig()

	// assign the delegate
	c.Delegate = d

	// Set the bind port
	c.BindPort = defaultPort

	// set the name
	c.Name = strings.Join([]string{"micro", hostname, uuid.New().String()}, "-")

	// TODO: set advertise addr to advertise behind nat

	// create the memberlist
	m, err := memberlist.Create(c)
	if err != nil {
		return err
	}

	// if we have memberlist join it
	if len(g.opts.Addrs) > 0 {
		_, err := m.Join(g.opts.Addrs)
		if err != nil {
			return err
		}
	}

	// Set the broadcast limit and number of nodes
	d.queue = &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return m.NumMembers()
		},
		RetransmitMult: 3,
	}

	g.memberlist = m
	g.delegate = d
	d.registry = g

	return nil
}

// NewRegistry returns a new gossip registry
func NewRegistry(opts ...registry.Option) registry.Registry {
	var options registry.Options
	for _, o := range opts {
		o(&options)
	}

	g := &gossipRegistry{
		opts: options,
	}
	if err := g.run(); err != nil {
		log.Fatal(err)
	}

	// return gossip registry
	return g
}
