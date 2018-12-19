// Package Gossip provides a gossip registry based on hashicorp/memberlist
package gossip

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/hashicorp/memberlist"
	log "github.com/micro/go-log"
	"github.com/micro/go-micro/registry"
	pb "github.com/micro/go-micro/registry/gossip/proto"
	"github.com/mitchellh/hashstructure"
)

const (
	addAction  = "update"
	delAction  = "delete"
	syncAction = "sync"
)

type broadcast struct {
	update *pb.Update
	notify chan<- struct{}
}

type delegate struct {
	queue   *memberlist.TransmitLimitedQueue
	updates chan *update
}

type gossipRegistry struct {
	queue    *memberlist.TransmitLimitedQueue
	updates  chan *update
	options  registry.Options
	member   *memberlist.Memberlist
	interval time.Duration

	sync.RWMutex
	services map[string][]*registry.Service

	s        sync.RWMutex
	watchers map[string]chan *registry.Result
}

type update struct {
	Update  *pb.Update
	Service *registry.Service
	sync    chan *registry.Service
}

var (
	// You should change this if using secure
	DefaultSecret = []byte("micro-gossip-key") // exactly 16 bytes
	ExpiryTick    = time.Second * 5
)

func configure(g *gossipRegistry, opts ...registry.Option) error {
	// loop through address list and get valid entries
	addrs := func(curAddrs []string) []string {
		var newAddrs []string
		for _, addr := range curAddrs {
			if trimAddr := strings.TrimSpace(addr); len(trimAddr) > 0 {
				newAddrs = append(newAddrs, trimAddr)
			}
		}
		return newAddrs
	}

	// current address list
	curAddrs := addrs(g.options.Addrs)

	// parse options
	for _, o := range opts {
		o(&g.options)
	}

	// new address list
	newAddrs := addrs(g.options.Addrs)

	// no new nodes and existing member. no configure
	if (len(newAddrs) == len(curAddrs)) && g.member != nil {
		return nil
	}

	// shutdown old member
	if g.member != nil {
		g.member.Shutdown()
	}

	// replace addresses
	curAddrs = newAddrs

	// create a queue
	queue := &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return len(curAddrs)
		},
		RetransmitMult: 3,
	}

	// create a new default config
	c := memberlist.DefaultLocalConfig()

	if optConfig, ok := g.options.Context.Value(contextConfig{}).(*memberlist.Config); ok && optConfig != nil {
		c = optConfig
	}

	if hostport, ok := g.options.Context.Value(contextAddress{}).(string); ok {
		host, port, err := net.SplitHostPort(hostport)
		if err == nil {
			pn, err := strconv.Atoi(port)
			if err == nil {
				c.BindPort = pn
			}
			c.BindAddr = host
		}
	} else {
		// set bind to random port
		c.BindPort = 0
	}

	if hostport, ok := g.options.Context.Value(contextAdvertise{}).(string); ok {
		host, port, err := net.SplitHostPort(hostport)
		if err == nil {
			pn, err := strconv.Atoi(port)
			if err == nil {
				c.AdvertisePort = pn
			}
			c.AdvertiseAddr = host
		}
	}

	// machine hostname
	hostname, _ := os.Hostname()

	// set the name
	c.Name = strings.Join([]string{"micro", hostname, uuid.New().String()}, "-")

	// set the delegate
	c.Delegate = &delegate{
		updates: g.updates,
		queue:   queue,
	}

	// log to dev null
	c.LogOutput = ioutil.Discard

	// set a secret key if secure
	if g.options.Secure {
		k, ok := g.options.Context.Value(contextSecretKey{}).([]byte)
		if !ok {
			// use the default secret
			k = DefaultSecret
		}
		c.SecretKey = k
	}

	// create the memberlist
	m, err := memberlist.Create(c)
	if err != nil {
		return err
	}

	// join the memberlist
	if len(curAddrs) > 0 {
		_, err := m.Join(curAddrs)
		if err != nil {
			return err
		}
	}

	// set internals
	g.queue = queue
	g.member = m
	g.interval = c.GossipInterval

	log.Logf("Registry Listening on %s", m.LocalNode().Address())
	return nil
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

	go func() {
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

		// send update
		d.updates <- &update{
			Update:  up,
			Service: service,
		}
	}()
}

func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.queue.GetBroadcasts(overhead, limit)
}

func (d *delegate) LocalState(join bool) []byte {
	if !join {
		return []byte{}
	}

	syncCh := make(chan *registry.Service, 1)
	services := map[string][]*registry.Service{}

	d.updates <- &update{
		Update: &pb.Update{
			Action: syncAction,
		},
		sync: syncCh,
	}

	for srv := range syncCh {
		services[srv.Name] = append(services[srv.Name], srv)
	}

	b, _ := json.Marshal(services)
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

	for _, service := range services {
		for _, srv := range service {
			d.updates <- &update{
				Update:  &pb.Update{Action: addAction},
				Service: srv,
				sync:    nil,
			}
		}
	}
}

func (g *gossipRegistry) publish(action string, services []*registry.Service) {
	g.s.RLock()
	for _, sub := range g.watchers {
		go func(sub chan *registry.Result) {
			for _, service := range services {
				sub <- &registry.Result{Action: action, Service: service}
			}
		}(sub)
	}
	g.s.RUnlock()
}

func (g *gossipRegistry) subscribe() (chan *registry.Result, chan bool) {
	next := make(chan *registry.Result, 10)
	exit := make(chan bool)

	id := uuid.New().String()

	g.s.Lock()
	g.watchers[id] = next
	g.s.Unlock()

	go func() {
		<-exit
		g.s.Lock()
		delete(g.watchers, id)
		close(next)
		g.s.Unlock()
	}()

	return next, exit
}

func (g *gossipRegistry) run() {
	var mtx sync.Mutex
	updates := map[uint64]*update{}

	// expiry loop
	go func() {
		t := time.NewTicker(ExpiryTick)
		defer t.Stop()

		for _ = range t.C {
			now := uint64(time.Now().UnixNano())

			mtx.Lock()

			// process all the updates
			for k, v := range updates {
				// check if expiry time has passed
				if d := (v.Update.Timestamp + v.Update.Expires); d < now {
					// delete from records
					delete(updates, k)
					// set to delete
					v.Update.Action = delAction
					// fire a new update
					g.updates <- v
				}
			}

			mtx.Unlock()
		}
	}()

	// process the updates
	for u := range g.updates {
		switch u.Update.Action {
		case addAction:
			g.Lock()
			if service, ok := g.services[u.Service.Name]; !ok {
				g.services[u.Service.Name] = []*registry.Service{u.Service}

			} else {
				g.services[u.Service.Name] = addServices(service, []*registry.Service{u.Service})
			}
			g.Unlock()

			// publish update to watchers
			go g.publish(addAction, []*registry.Service{u.Service})

			// we need to expire the node at some point in the future
			if u.Update.Expires > 0 {
				// create a hash of this service
				if hash, err := hashstructure.Hash(u.Service, nil); err == nil {
					mtx.Lock()
					updates[hash] = u
					mtx.Unlock()
				}
			}
		case delAction:
			g.Lock()
			if service, ok := g.services[u.Service.Name]; ok {
				if services := delServices(service, []*registry.Service{u.Service}); len(services) == 0 {
					delete(g.services, u.Service.Name)
				} else {
					g.services[u.Service.Name] = services
				}
			}
			g.Unlock()

			// publish update to watchers
			go g.publish(delAction, []*registry.Service{u.Service})

			// delete from expiry checks
			if hash, err := hashstructure.Hash(u.Service, nil); err == nil {
				mtx.Lock()
				delete(updates, hash)
				mtx.Unlock()
			}
		case syncAction:
			// no sync channel provided
			if u.sync == nil {
				continue
			}

			g.RLock()

			// push all services through the sync chan
			for _, service := range g.services {
				for _, srv := range service {
					u.sync <- srv
				}

				// publish to watchers
				go g.publish(addAction, service)
			}

			g.RUnlock()

			// close the sync chan
			close(u.sync)
		}
	}
}

func (g *gossipRegistry) Init(opts ...registry.Option) error {
	return configure(g, opts...)
}

func (g *gossipRegistry) Options() registry.Options {
	return g.options
}

func (g *gossipRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}

	g.Lock()
	if service, ok := g.services[s.Name]; !ok {
		g.services[s.Name] = []*registry.Service{s}
	} else {
		g.services[s.Name] = addServices(service, []*registry.Service{s})
	}
	g.Unlock()

	var options registry.RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	up := &pb.Update{
		Id:        uuid.New().String(),
		Timestamp: uint64(time.Now().UnixNano()),
		Expires:   uint64(options.TTL.Nanoseconds()),
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

	// wait
	<-time.After(g.interval * 2)

	return nil
}

func (g *gossipRegistry) Deregister(s *registry.Service) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}

	g.Lock()
	if service, ok := g.services[s.Name]; ok {
		if services := delServices(service, []*registry.Service{s}); len(services) == 0 {
			delete(g.services, s.Name)
		} else {
			g.services[s.Name] = services
		}
	}
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

	// wait
	<-time.After(g.interval * 2)

	return nil
}

func (g *gossipRegistry) GetService(name string) ([]*registry.Service, error) {
	g.RLock()
	service, ok := g.services[name]
	g.RUnlock()
	if !ok {
		return nil, registry.ErrNotFound
	}
	return service, nil
}

func (g *gossipRegistry) ListServices() ([]*registry.Service, error) {
	var services []*registry.Service
	g.RLock()
	for _, service := range g.services {
		services = append(services, service...)
	}
	g.RUnlock()
	return services, nil
}

func (g *gossipRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	n, e := g.subscribe()
	return newGossipWatcher(n, e, opts...)
}

func (g *gossipRegistry) String() string {
	return "gossip"
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	gossip := &gossipRegistry{
		options: registry.Options{
			Context: context.Background(),
		},
		updates:  make(chan *update, 100),
		services: make(map[string][]*registry.Service),
		watchers: make(map[string]chan *registry.Result),
	}

	// run the updater
	go gossip.run()

	// configure the gossiper
	if err := configure(gossip, opts...); err != nil {
		log.Fatalf("Error configuring registry: %v", err)
	}

	// wait for setup
	<-time.After(gossip.interval * 2)

	return gossip
}
