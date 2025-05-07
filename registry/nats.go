//go:build nats
// +build nats

// Package nats provides a NATS registry using broadcast queries
package registry

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type natsRegistry struct {
	addrs          []string
	opts           Options
	nopts          nats.Options
	queryTopic     string
	watchTopic     string
	registerAction string

	sync.RWMutex
	conn      *nats.Conn
	services  map[string][]*Service
	listeners map[string]chan bool
}

var (
	defaultQueryTopic     = "micro.nats.query"
	defaultWatchTopic     = "micro.nats.watch"
	defaultRegisterAction = "create"
)

func configure(n *natsRegistry, opts ...Option) error {
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

	registerAction := defaultRegisterAction
	if ra, ok := n.opts.Context.Value(registerActionKey{}).(string); ok {
		registerAction = ra
	}

	// Options have higher priority than nats.Options
	// only if Addrs, Secure or TLSConfig were not set through a Option
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
	// stored in natsaddrs and n.opts.Addrs are identical)
	n.opts.Addrs = setAddrs(n.opts.Addrs)

	n.addrs = n.opts.Addrs
	n.nopts = natsOptions
	n.queryTopic = queryTopic
	n.watchTopic = watchTopic
	n.registerAction = registerAction

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

func (n *natsRegistry) register(s *Service) error {
	conn, err := n.getConn()
	if err != nil {
		return err
	}

	n.Lock()
	defer n.Unlock()

	// cache service
	n.services[s.Name] = addServices(n.services[s.Name], cp([]*Service{s}))

	// create query listener
	if n.listeners[s.Name] == nil {
		listener := make(chan bool)

		// create a subscriber that responds to queries
		sub, err := conn.Subscribe(n.queryTopic, func(m *nats.Msg) {
			var result *Result

			if err := json.Unmarshal(m.Data, &result); err != nil {
				return
			}

			var services []*Service

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

func (n *natsRegistry) deregister(s *Service) error {
	n.Lock()
	defer n.Unlock()

	services := delServices(n.services[s.Name], cp([]*Service{s}))
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

func (n *natsRegistry) query(s string, quorum int) ([]*Service, error) {
	conn, err := n.getConn()
	if err != nil {
		return nil, err
	}

	var action string
	var service *Service

	if len(s) > 0 {
		action = "get"
		service = &Service{Name: s}
	} else {
		action = "list"
	}

	inbox := nats.NewInbox()

	response := make(chan *Service, 10)

	sub, err := conn.Subscribe(inbox, func(m *nats.Msg) {
		var service *Service
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

	b, err := json.Marshal(&Result{Action: action, Service: service})
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

	serviceMap := make(map[string]*Service)

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

	var services []*Service
	for _, service := range serviceMap {
		services = append(services, service)
	}
	return services, nil
}

func (n *natsRegistry) Init(opts ...Option) error {
	return configure(n, opts...)
}

func (n *natsRegistry) Options() Options {
	return n.opts
}

func (n *natsRegistry) Register(s *Service, opts ...RegisterOption) error {
	if err := n.register(s); err != nil {
		return err
	}

	conn, err := n.getConn()
	if err != nil {
		return err
	}

	b, err := json.Marshal(&Result{Action: n.registerAction, Service: s})
	if err != nil {
		return err
	}

	return conn.Publish(n.watchTopic, b)
}

func (n *natsRegistry) Deregister(s *Service, opts ...DeregisterOption) error {
	if err := n.deregister(s); err != nil {
		return err
	}

	conn, err := n.getConn()
	if err != nil {
		return err
	}

	b, err := json.Marshal(&Result{Action: "delete", Service: s})
	if err != nil {
		return err
	}
	return conn.Publish(n.watchTopic, b)
}

func (n *natsRegistry) GetService(s string, opts ...GetOption) ([]*Service, error) {
	services, err := n.query(s, getQuorum(n.opts))
	if err != nil {
		return nil, err
	}
	return services, nil
}

func (n *natsRegistry) ListServices(opts ...ListOption) ([]*Service, error) {
	s, err := n.query("", 0)
	if err != nil {
		return nil, err
	}

	var services []*Service
	serviceMap := make(map[string]*Service)

	for _, v := range s {
		serviceMap[v.Name] = &Service{Name: v.Name, Version: v.Version}
	}

	for _, v := range serviceMap {
		services = append(services, v)
	}

	return services, nil
}

func (n *natsRegistry) Watch(opts ...WatchOption) (Watcher, error) {
	conn, err := n.getConn()
	if err != nil {
		return nil, err
	}

	sub, err := conn.SubscribeSync(n.watchTopic)
	if err != nil {
		return nil, err
	}

	var wo WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	return &natsWatcher{sub, wo}, nil
}

func (n *natsRegistry) String() string {
	return "nats"
}

func NewRegistry(opts ...Option) Registry {
	options := Options{
		Timeout: time.Millisecond * 100,
		Context: context.Background(),
	}

	n := &natsRegistry{
		opts:      options,
		services:  make(map[string][]*Service),
		listeners: make(map[string]chan bool),
	}
	configure(n, opts...)
	return n
}
