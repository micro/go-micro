// Package nats provides a NATS registry using broadcast queries
package nats

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/registry"
	"github.com/nats-io/nats.go"
)

type natsRegistry struct {
	addrs      []string
	opts       registry.Options
	nopts      nats.Options
	queryTopic string
	watchTopic string

	sync.RWMutex
	conn      *nats.Conn
	services  map[string][]*registry.Service
	listeners map[string]chan bool
}

var (
	defaultQueryTopic = "micro.registry.nats.query"
	defaultWatchTopic = "micro.registry.nats.watch"
)

func init() {
	cmd.DefaultRegistries["nats"] = NewRegistry
}

func configure(n *natsRegistry, opts ...registry.Option) error {
	for _, o := range opts {
		o(&n.opts)
	}

	natsOptions := nats.GetDefaultOptions()
	if n, ok := n.opts.Context.Value(optionsKey{}).(nats.Options); ok {
		natsOptions = n
	}

	queryTopic := defaultQueryTopic
	if qt, ok := n.opts.Context.Value(queryTopicKey{}).(string); ok {
		queryTopic = qt
	}

	watchTopic := defaultWatchTopic
	if wt, ok := n.opts.Context.Value(watchTopicKey{}).(string); ok {
		watchTopic = wt
	}

	// registry.Options have higher priority than nats.Options
	// only if Addrs, Secure or TLSConfig were not set through a registry.Option
	// we read them from nats.Option
	if len(n.opts.Addrs) == 0 {
		n.opts.Addrs = natsOptions.Servers
	}

	if !n.opts.Secure {
		n.opts.Secure = natsOptions.Secure
	}

	if n.opts.TLSConfig == nil {
		n.opts.TLSConfig = natsOptions.TLSConfig
	}

	// check & add nats:// prefix (this makes also sure that the addresses
	// stored in natsRegistry.addrs and n.opts.Addrs are identical)
	n.opts.Addrs = setAddrs(n.opts.Addrs)

	n.addrs = n.opts.Addrs
	n.nopts = natsOptions
	n.queryTopic = queryTopic
	n.watchTopic = watchTopic

	return nil
}
func setAddrs(addrs []string) []string {
	var cAddrs []string
	for _, addr := range addrs {
		if len(addr) == 0 {
			continue
		}
		if !strings.HasPrefix(addr, "nats://") {
			addr = "nats://" + addr
		}
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{nats.DefaultURL}
	}
	return cAddrs
}

func (n *natsRegistry) newConn() (*nats.Conn, error) {
	opts := n.nopts
	opts.Servers = n.addrs
	opts.Secure = n.opts.Secure
	opts.TLSConfig = n.opts.TLSConfig

	// secure might not be set
	if opts.TLSConfig != nil {
		opts.Secure = true
	}

	return opts.Connect()
}

func (n *natsRegistry) getConn() (*nats.Conn, error) {
	n.Lock()
	defer n.Unlock()

	if n.conn != nil {
		return n.conn, nil
	}

	c, err := n.newConn()
	if err != nil {
		return nil, err
	}
	n.conn = c

	return n.conn, nil
}

func (n *natsRegistry) register(s *registry.Service) error {
	conn, err := n.getConn()
	if err != nil {
		return err
	}

	n.Lock()
	defer n.Unlock()

	// cache service
	n.services[s.Name] = addServices(n.services[s.Name], cp([]*registry.Service{s}))

	// create query listener
	if n.listeners[s.Name] == nil {
		listener := make(chan bool)

		// create a subscriber that responds to queries
		sub, err := conn.Subscribe(n.queryTopic, func(m *nats.Msg) {
			var result *registry.Result

			if err := json.Unmarshal(m.Data, &result); err != nil {
				return
			}

			var services []*registry.Service

			switch result.Action {
			// is this a get query and we own the service?
			case "get":
				if result.Service.Name != s.Name {
					return
				}
				n.RLock()
				services = cp(n.services[s.Name])
				n.RUnlock()
			// it's a list request, but we're still only a
			// subscriber for this service... so just get this service
			// totally suboptimal
			case "list":
				n.RLock()
				services = cp(n.services[s.Name])
				n.RUnlock()
			default:
				// does not match
				return
			}

			// respond to query
			for _, service := range services {
				b, err := json.Marshal(service)
				if err != nil {
					continue
				}
				conn.Publish(m.Reply, b)
			}
		})
		if err != nil {
			return err
		}

		// Unsubscribe if we're told to do so
		go func() {
			<-listener
			sub.Unsubscribe()
		}()

		n.listeners[s.Name] = listener
	}

	return nil
}

func (n *natsRegistry) deregister(s *registry.Service) error {
	n.Lock()
	defer n.Unlock()

	// cache leftover service
	services := addServices(n.services[s.Name], cp([]*registry.Service{s}))
	if len(services) > 0 {
		n.services[s.Name] = services
		return nil
	}

	// delete cached service
	delete(n.services, s.Name)

	// delete query listener
	if listener, lexists := n.listeners[s.Name]; lexists {
		close(listener)
		delete(n.listeners, s.Name)
	}

	return nil
}

func (n *natsRegistry) query(s string, quorum int) ([]*registry.Service, error) {
	conn, err := n.getConn()
	if err != nil {
		return nil, err
	}

	var action string
	var service *registry.Service

	if len(s) > 0 {
		action = "get"
		service = &registry.Service{Name: s}
	} else {
		action = "list"
	}

	inbox := nats.NewInbox()

	response := make(chan *registry.Service, 10)

	sub, err := conn.Subscribe(inbox, func(m *nats.Msg) {
		var service *registry.Service
		if err := json.Unmarshal(m.Data, &service); err != nil {
			return
		}
		select {
		case response <- service:
		case <-time.After(n.opts.Timeout):
		}
	})
	if err != nil {
		return nil, err
	}
	defer sub.Unsubscribe()

	b, err := json.Marshal(&registry.Result{Action: action, Service: service})
	if err != nil {
		return nil, err
	}

	if err := conn.PublishMsg(&nats.Msg{
		Subject: n.queryTopic,
		Reply:   inbox,
		Data:    b,
	}); err != nil {
		return nil, err
	}

	timeoutChan := time.After(n.opts.Timeout)

	serviceMap := make(map[string]*registry.Service)

loop:
	for {
		select {
		case service := <-response:
			key := service.Name + "-" + service.Version
			srv, ok := serviceMap[key]
			if ok {
				srv.Nodes = append(srv.Nodes, service.Nodes...)
				serviceMap[key] = srv
			} else {
				serviceMap[key] = service
			}

			if quorum > 0 && len(serviceMap[key].Nodes) >= quorum {
				break loop
			}
		case <-timeoutChan:
			break loop
		}
	}

	var services []*registry.Service
	for _, service := range serviceMap {
		services = append(services, service)
	}
	return services, nil
}

func (n *natsRegistry) Init(opts ...registry.Option) error {
	return configure(n, opts...)
}

func (n *natsRegistry) Options() registry.Options {
	return n.opts
}

func (n *natsRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	if err := n.register(s); err != nil {
		return err
	}

	conn, err := n.getConn()
	if err != nil {
		return err
	}

	b, err := json.Marshal(&registry.Result{Action: "create", Service: s})
	if err != nil {
		return err
	}

	return conn.Publish(n.watchTopic, b)
}

func (n *natsRegistry) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {
	if err := n.deregister(s); err != nil {
		return err
	}

	conn, err := n.getConn()
	if err != nil {
		return err
	}

	b, err := json.Marshal(&registry.Result{Action: "delete", Service: s})
	if err != nil {
		return err
	}
	return conn.Publish(n.watchTopic, b)
}

func (n *natsRegistry) GetService(s string, opts ...registry.GetOption) ([]*registry.Service, error) {
	services, err := n.query(s, getQuorum(n.opts))
	if err != nil {
		return nil, err
	}
	return services, nil
}

func (n *natsRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	s, err := n.query("", 0)
	if err != nil {
		return nil, err
	}

	var services []*registry.Service
	serviceMap := make(map[string]*registry.Service)

	for _, v := range s {
		serviceMap[v.Name] = &registry.Service{Name: v.Name}
	}

	for _, v := range serviceMap {
		services = append(services, v)
	}

	return services, nil
}

func (n *natsRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	conn, err := n.getConn()
	if err != nil {
		return nil, err
	}

	sub, err := conn.SubscribeSync(n.watchTopic)
	if err != nil {
		return nil, err
	}

	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	return &natsWatcher{sub, wo}, nil
}

func (n *natsRegistry) String() string {
	return "nats"
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	options := registry.Options{
		Timeout: time.Millisecond * 100,
		Context: context.Background(),
	}

	n := &natsRegistry{
		opts:      options,
		services:  make(map[string][]*registry.Service),
		listeners: make(map[string]chan bool),
	}
	configure(n, opts...)
	return n
}
