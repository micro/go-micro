package server

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/myodc/go-micro/broker"
	c "github.com/myodc/go-micro/context"
	"github.com/myodc/go-micro/registry"
	"github.com/myodc/go-micro/transport"

	log "github.com/golang/glog"
	rpc "github.com/youtube/vitess/go/rpcplus"

	"golang.org/x/net/context"
)

type rpcServer struct {
	rpc  *rpc.Server
	exit chan chan error

	sync.RWMutex
	opts        options
	handlers    map[string]Handler
	subscribers map[*subscriber][]broker.Subscriber
}

func newRpcServer(opts ...Option) Server {
	return &rpcServer{
		opts:        newOptions(opts...),
		rpc:         rpc.NewServer(),
		handlers:    make(map[string]Handler),
		subscribers: make(map[*subscriber][]broker.Subscriber),
		exit:        make(chan chan error),
	}
}

func (s *rpcServer) accept(sock transport.Socket) {
	var msg transport.Message
	if err := sock.Recv(&msg); err != nil {
		return
	}

	codec := newRpcPlusCodec(&msg, sock)

	// strip our headers
	hdr := make(map[string]string)
	for k, v := range msg.Header {
		hdr[k] = v
	}
	delete(hdr, "Content-Type")

	ctx := c.WithMetadata(context.Background(), hdr)
	s.rpc.ServeRequestWithContext(ctx, codec)
}

func (s *rpcServer) Config() options {
	s.RLock()
	opts := s.opts
	s.RUnlock()
	return opts
}

func (s *rpcServer) Init(opts ...Option) {
	s.Lock()
	for _, opt := range opts {
		opt(&s.opts)
	}
	if len(s.opts.id) == 0 {
		s.opts.id = s.opts.name + "-" + DefaultId
	}
	s.Unlock()
}

func (s *rpcServer) NewHandler(h interface{}) Handler {
	return newRpcHandler(h)
}

func (s *rpcServer) Handle(h Handler) error {
	if err := s.rpc.Register(h.Handler()); err != nil {
		return err
	}
	s.Lock()
	s.handlers[h.Name()] = h
	s.Unlock()
	return nil
}

func (s *rpcServer) NewSubscriber(topic string, sb interface{}) Subscriber {
	return newSubscriber(topic, sb)
}

func (s *rpcServer) Subscribe(sb Subscriber) error {
	sub, ok := sb.(*subscriber)
	if !ok {
		return fmt.Errorf("invalid subscriber: expected *subscriber")
	}
	if len(sub.handlers) == 0 {
		return fmt.Errorf("invalid subscriber: no handler functions")
	}

	s.Lock()
	_, ok = s.subscribers[sub]
	if ok {
		return fmt.Errorf("subscriber %v already exists", s)
	}
	s.subscribers[sub] = nil
	s.Unlock()
	return nil
}

func (s *rpcServer) Register() error {
	// parse address for host, port
	config := s.Config()
	var advt, host string
	var port int

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(config.Advertise()) > 0 {
		advt = config.Advertise()
	} else {
		advt = config.Address()
	}

	parts := strings.Split(advt, ":")
	if len(parts) > 1 {
		host = strings.Join(parts[:len(parts)-1], ":")
		port, _ = strconv.Atoi(parts[len(parts)-1])
	} else {
		host = parts[0]
	}

	addr, err := extractAddress(host)
	if err != nil {
		return err
	}

	// register service
	node := &registry.Node{
		Id:       config.Id(),
		Address:  addr,
		Port:     port,
		Metadata: config.Metadata(),
	}

	s.RLock()
	var endpoints []*registry.Endpoint
	for _, e := range s.handlers {
		endpoints = append(endpoints, e.Endpoints()...)
	}
	for e, _ := range s.subscribers {
		endpoints = append(endpoints, e.Endpoints()...)
	}
	s.RUnlock()

	service := &registry.Service{
		Name:      config.Name(),
		Version:   config.Version(),
		Nodes:     []*registry.Node{node},
		Endpoints: endpoints,
	}

	log.Infof("Registering node: %s", node.Id)
	if err := config.registry.Register(service); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	for sb, _ := range s.subscribers {
		handler := createSubHandler(sb)
		sub, err := config.broker.Subscribe(sb.Topic(), handler)
		if err != nil {
			return err
		}
		s.subscribers[sb] = []broker.Subscriber{sub}
	}

	return nil
}

func (s *rpcServer) Deregister() error {
	config := s.Config()
	var advt, host string
	var port int

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(config.Advertise()) > 0 {
		advt = config.Advertise()
	} else {
		advt = config.Address()
	}

	parts := strings.Split(advt, ":")
	if len(parts) > 1 {
		host = strings.Join(parts[:len(parts)-1], ":")
		port, _ = strconv.Atoi(parts[len(parts)-1])
	} else {
		host = parts[0]
	}

	addr, err := extractAddress(host)
	if err != nil {
		return err
	}

	node := &registry.Node{
		Id:      config.Id(),
		Address: addr,
		Port:    port,
	}

	service := &registry.Service{
		Name:    config.Name(),
		Version: config.Version(),
		Nodes:   []*registry.Node{node},
	}

	log.Infof("Deregistering node: %s", node.Id)
	if err := config.registry.Deregister(service); err != nil {
		return err
	}

	s.Lock()
	for sb, subs := range s.subscribers {
		for _, sub := range subs {
			log.Infof("Unsubscribing from topic: %s", sub.Topic())
			sub.Unsubscribe()
		}
		s.subscribers[sb] = nil
	}
	s.Unlock()
	return nil
}

func (s *rpcServer) Start() error {
	registerHealthChecker(s)
	config := s.Config()

	ts, err := config.transport.Listen(s.opts.address)
	if err != nil {
		return err
	}

	log.Infof("Listening on %s", ts.Addr())
	s.Lock()
	s.opts.address = ts.Addr()
	s.Unlock()

	go ts.Accept(s.accept)

	go func() {
		ch := <-s.exit
		ch <- ts.Close()
		config.broker.Disconnect()
	}()

	// TODO: subscribe to cruft
	return config.broker.Connect()
}

func (s *rpcServer) Stop() error {
	ch := make(chan error)
	s.exit <- ch
	return <-ch
}
