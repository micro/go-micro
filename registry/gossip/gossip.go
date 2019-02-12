// Package Gossip provides a gossip registry based on hashicorp/memberlist
package gossip

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/hashicorp/memberlist"
	log "github.com/micro/go-log"
	"github.com/micro/go-micro/registry"
	pb "github.com/micro/go-micro/registry/gossip/proto"
	"github.com/mitchellh/hashstructure"
)

// use registry.Result int32 values after it switches from string to int32 types
// type actionType int32
// type updateType int32

const (
	actionTypeInvalid int32 = iota
	actionTypeCreate
	actionTypeDelete
	actionTypeUpdate
	actionTypeSync
)

const (
	nodeActionUnknown int32 = iota
	nodeActionJoin
	nodeActionLeave
	nodeActionUpdate
)

func actionTypeString(t int32) string {
	switch t {
	case actionTypeCreate:
		return "create"
	case actionTypeDelete:
		return "delete"
	case actionTypeUpdate:
		return "update"
	case actionTypeSync:
		return "sync"
	}
	return "invalid"
}

const (
	updateTypeInvalid int32 = iota
	updateTypeService
)

type broadcast struct {
	update *pb.Update
	notify chan<- struct{}
}

type delegate struct {
	queue   *memberlist.TransmitLimitedQueue
	updates chan *update
}

type event struct {
	action int32
	node   string
}

type eventDelegate struct {
	events chan *event
}

func (ed *eventDelegate) NotifyJoin(n *memberlist.Node) {
	ed.events <- &event{action: nodeActionJoin, node: n.Address()}
}
func (ed *eventDelegate) NotifyLeave(n *memberlist.Node) {
	ed.events <- &event{action: nodeActionLeave, node: n.Address()}
}
func (ed *eventDelegate) NotifyUpdate(n *memberlist.Node) {
	ed.events <- &event{action: nodeActionUpdate, node: n.Address()}
}

type gossipRegistry struct {
	queue       *memberlist.TransmitLimitedQueue
	updates     chan *update
	events      chan *event
	options     registry.Options
	member      *memberlist.Memberlist
	interval    time.Duration
	tcpInterval time.Duration

	connectRetry   bool
	connectTimeout time.Duration
	sync.RWMutex
	services map[string][]*registry.Service

	watchers map[string]chan *registry.Result

	mtu     int
	addrs   []string
	members map[string]int32
	done    chan struct{}
}

type update struct {
	Update  *pb.Update
	Service *registry.Service
	sync    chan *registry.Service
}

var (
	// You should change this if using secure
	DefaultSecret = []byte("micro-gossip-key") // exactly 16 bytes
	ExpiryTick    = time.Second * 1            // needs to be smaller than registry.RegisterTTL
	MaxPacketSize = 512
)

func (g *gossipRegistry) connect(addrs []string) error {
	var err error

	if len(addrs) == 0 {
		return nil
	}

	timeout := make(<-chan time.Time)
	if g.connectTimeout > 0 {
		timeout = time.After(g.connectTimeout)
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	fn := func() (int, error) {
		return g.member.Join(addrs)
	}

	// don't wait for first try
	if _, err = fn(); err == nil {
		return nil
	}

	// wait loop
	for {
		select {
		// context closed
		case <-g.options.Context.Done():
			return nil
		// call close, don't wait anymore
		case <-g.done:
			return nil
		//  in case of timeout fail with a timeout error
		case <-timeout:
			return fmt.Errorf("[gossip]: timedout connect to %v", g.addrs)
		// got a tick, try to connect
		case <-ticker.C:
			if _, err = fn(); err == nil {
				log.Logf("[gossip]: success connect to %v", g.addrs)
				return nil
			} else {
				log.Logf("[gossip]: failed connect to %v", g.addrs)
			}
		}
	}

	return err
}

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
		g.Stop()
	}

	// replace addresses
	curAddrs = newAddrs

	// create a new default config
	c := memberlist.DefaultLocalConfig()

	// sane good default options
	c.LogOutput = ioutil.Discard // log to /dev/null
	c.PushPullInterval = 0       // disable expensive tcp push/pull
	c.ProtocolVersion = 4        // suport latest stable features

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

	// set a secret key if secure
	if g.options.Secure {
		k, ok := g.options.Context.Value(contextSecretKey{}).([]byte)
		if !ok {
			// use the default secret
			k = DefaultSecret
		}
		c.SecretKey = k
	}

	if v, ok := g.options.Context.Value(connectRetry{}).(bool); ok && v {
		g.connectRetry = true
	}
	if td, ok := g.options.Context.Value(connectTimeout{}).(time.Duration); ok {
		g.connectTimeout = td
	}

	// create a queue
	queue := &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return len(curAddrs)
		},
		RetransmitMult: 3,
	}

	// set the delegate
	c.Delegate = &delegate{
		updates: g.updates,
		queue:   queue,
	}

	if g.connectRetry {
		c.Events = &eventDelegate{
			events: g.events,
		}
	}

	// create the memberlist
	m, err := memberlist.Create(c)
	if err != nil {
		return err
	}

	// set internals
	g.Lock()
	if len(curAddrs) > 0 {
		for _, addr := range curAddrs {
			g.members[addr] = nodeActionUnknown
		}
	}
	g.tcpInterval = c.PushPullInterval
	g.addrs = curAddrs
	g.queue = queue
	g.member = m
	g.interval = c.GossipInterval
	g.Unlock()

	log.Logf("[gossip]: Registry Listening on %s", m.LocalNode().Address())
	return g.connect(curAddrs)
}

func (*broadcast) UniqueBroadcast() {}

func (b *broadcast) Invalidates(other memberlist.Broadcast) bool {
	return false
}

func (b *broadcast) Message() []byte {
	up, err := proto.Marshal(b.update)
	if err != nil {
		return nil
	}
	if l := len(up); l > MaxPacketSize {
		log.Logf("[gossip]: broadcast message size %d bigger then MaxPacketSize %d", l, MaxPacketSize)
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
		if up.Type != updateTypeService {
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
			Action: actionTypeSync,
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
				Update:  &pb.Update{Action: actionTypeCreate},
				Service: srv,
				sync:    nil,
			}
		}
	}
}

func (g *gossipRegistry) publish(action string, services []*registry.Service) {
	g.RLock()
	for _, sub := range g.watchers {
		go func(sub chan *registry.Result) {
			for _, service := range services {
				sub <- &registry.Result{Action: action, Service: service}
			}
		}(sub)
	}
	g.RUnlock()
}

func (g *gossipRegistry) subscribe() (chan *registry.Result, chan bool) {
	next := make(chan *registry.Result, 10)
	exit := make(chan bool)

	id := uuid.New().String()

	g.Lock()
	g.watchers[id] = next
	g.Unlock()

	go func() {
		<-exit
		g.Lock()
		delete(g.watchers, id)
		close(next)
		g.Unlock()
	}()

	return next, exit
}

func (g *gossipRegistry) wait() {
	ctx := g.options.Context

	if c, ok := ctx.Value(contextContext{}).(context.Context); ok && c != nil {
		ctx = c
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	select {
	// wait on kill signal
	case <-ch:
	// wait on context cancel
	case <-ctx.Done():
	}

	g.Stop()
}

func (g *gossipRegistry) Stop() error {
	g.Lock()
	if g.done != nil {
		close(g.done)
		g.done = nil
	}
	if g.member != nil {
		g.member.Leave(g.interval * 2)
		g.member.Shutdown()
		g.member = nil
	}
	g.Unlock()
	return nil
}

func (g *gossipRegistry) run() {
	var mtx sync.Mutex
	updates := map[uint64]*update{}

	// expiry loop
	go func() {
		ticker := time.NewTicker(ExpiryTick)
		defer ticker.Stop()

		for {
			select {
			case <-g.done:
				return
			case <-ticker.C:
				now := uint64(time.Now().UnixNano())

				mtx.Lock()

				// process all the updates
				for k, v := range updates {
					// check if expiry time has passed
					if d := (v.Update.Expires); d < now {
						// delete from records
						delete(updates, k)
						// set to delete
						v.Update.Action = actionTypeDelete
						// fire a new update
						g.updates <- v
					}
				}

				mtx.Unlock()
			}
		}
	}()

	go func() {
		for {
			select {
			case <-g.done:
				return
			case ed := <-g.events:
				// may be not block all registry?
				g.Lock()
				if _, ok := g.members[ed.node]; ok {
					g.members[ed.node] = ed.action
				}
				g.Unlock()
			}
		}
	}()

	if g.connectRetry {
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-g.done:
					return
				case <-ticker.C:
					var addrs []string
					g.RLock()
					for node, action := range g.members {
						if action == nodeActionLeave && g.member.LocalNode().Address() != node {
							addrs = append(addrs, node)
						}
					}
					g.RUnlock()
					if len(addrs) > 0 {
						g.connect(addrs)
					}
				}
			}
		}()
	}

	// process the updates
	for u := range g.updates {
		switch u.Update.Action {
		case actionTypeCreate:
			g.Lock()
			if service, ok := g.services[u.Service.Name]; !ok {
				g.services[u.Service.Name] = []*registry.Service{u.Service}

			} else {
				g.services[u.Service.Name] = addServices(service, []*registry.Service{u.Service})
			}
			g.Unlock()

			// publish update to watchers
			go g.publish(actionTypeString(actionTypeCreate), []*registry.Service{u.Service})

			// we need to expire the node at some point in the future
			if u.Update.Expires > 0 {
				// create a hash of this service
				if hash, err := hashstructure.Hash(u.Service, nil); err == nil {
					mtx.Lock()
					updates[hash] = u
					mtx.Unlock()
				}
			}
		case actionTypeDelete:
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
			go g.publish(actionTypeString(actionTypeDelete), []*registry.Service{u.Service})

			// delete from expiry checks
			if hash, err := hashstructure.Hash(u.Service, nil); err == nil {
				mtx.Lock()
				delete(updates, hash)
				mtx.Unlock()
			}
		case actionTypeSync:
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
				go g.publish(actionTypeString(actionTypeCreate), service)
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

	if options.TTL == 0 && g.tcpInterval == 0 {
		return fmt.Errorf("must provide registry.RegisterTTL option or set PushPullInterval in *memberlist.Config")
	}

	up := &pb.Update{
		Expires: uint64(time.Now().Add(options.TTL).UnixNano()),
		Action:  actionTypeCreate,
		Type:    updateTypeService,
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
		Action: actionTypeDelete,
		Type:   updateTypeService,
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
		events:   make(chan *event, 100),
		services: make(map[string][]*registry.Service),
		watchers: make(map[string]chan *registry.Result),
		done:     make(chan struct{}),
		members:  make(map[string]int32),
	}

	// run the updater
	go gossip.run()

	// configure the gossiper
	if err := configure(gossip, opts...); err != nil {
		log.Fatalf("Error configuring registry: %v", err)
	}

	// wait for setup
	<-time.After(gossip.interval * 2)

	go gossip.wait()

	return gossip
}
