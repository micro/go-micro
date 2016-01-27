package server

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/codec"
	c "github.com/micro/go-micro/context"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/transport"

	log "github.com/golang/glog"

	"golang.org/x/net/context"
)

type rpcServer struct {
	rpc  *server
	exit chan chan error

	sync.RWMutex
	opts        Options
	handlers    map[string]Handler
	subscribers map[*subscriber][]broker.Subscriber
}

func newRpcServer(opts ...Option) Server {
	options := newOptions(opts...)
	return &rpcServer{
		opts: options,
		rpc: &server{
			name:         options.Name,
			serviceMap:   make(map[string]*service),
			hdlrWrappers: options.HdlrWrappers,
		},
		handlers:    make(map[string]Handler),
		subscribers: make(map[*subscriber][]broker.Subscriber),
		exit:        make(chan chan error),
	}
}

func (s *rpcServer) accept(sock transport.Socket) {
	defer func() {
		if r := recover(); r != nil {
			log.Error(r, string(debug.Stack()))
			sock.Close()
		}
	}()

	var msg transport.Message
	if err := sock.Recv(&msg); err != nil {
		return
	}

	ct := msg.Header["Content-Type"]
	cf, err := s.newCodec(ct)
	// TODO: needs better error handling
	if err != nil {
		sock.Send(&transport.Message{
			Header: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: []byte(err.Error()),
		})
		sock.Close()
		return
	}

	codec := newRpcPlusCodec(&msg, sock, cf)

	// strip our headers
	hdr := make(map[string]string)
	for k, v := range msg.Header {
		hdr[k] = v
	}
	delete(hdr, "Content-Type")

	ctx := c.WithMetadata(context.Background(), hdr)

	// TODO: needs better error handling
	if err := s.rpc.serveRequest(ctx, codec, ct); err != nil {
		log.Errorf("Unexpected error serving request, closing socket: %v", err)
		sock.Close()
	}
}

func (s *rpcServer) newCodec(contentType string) (codec.NewCodec, error) {
	if cf, ok := s.opts.Codecs[contentType]; ok {
		return cf, nil
	}
	if cf, ok := defaultCodecs[contentType]; ok {
		return cf, nil
	}
	return nil, fmt.Errorf("Unsupported Content-Type: %s", contentType)
}

func (s *rpcServer) Options() Options {
	s.RLock()
	opts := s.opts
	s.RUnlock()
	return opts
}

func (s *rpcServer) Init(opts ...Option) error {
	s.Lock()
	for _, opt := range opts {
		opt(&s.opts)
	}
	s.Unlock()
	return nil
}

func (s *rpcServer) NewHandler(h interface{}, opts ...HandlerOption) Handler {
	return newRpcHandler(h, opts...)
}

func (s *rpcServer) Handle(h Handler) error {
	if err := s.rpc.register(h.Handler()); err != nil {
		return err
	}
	s.Lock()
	s.handlers[h.Name()] = h
	s.Unlock()
	return nil
}

func (s *rpcServer) NewSubscriber(topic string, sb interface{}, opts ...SubscriberOption) Subscriber {
	return newSubscriber(topic, sb, opts...)
}

func (s *rpcServer) Subscribe(sb Subscriber) error {
	sub, ok := sb.(*subscriber)
	if !ok {
		return fmt.Errorf("invalid subscriber: expected *subscriber")
	}
	if len(sub.handlers) == 0 {
		return fmt.Errorf("invalid subscriber: no handler functions")
	}

	if err := validateSubscriber(sb); err != nil {
		return err
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
	config := s.Options()
	var advt, host string
	var port int

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(config.Advertise) > 0 {
		advt = config.Advertise
	} else {
		advt = config.Address
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
		Id:       config.Name + "-" + config.Id,
		Address:  addr,
		Port:     port,
		Metadata: config.Metadata,
	}

	node.Metadata["transport"] = config.Transport.String()
	node.Metadata["broker"] = config.Broker.String()
	node.Metadata["server"] = s.String()
	node.Metadata["registry"] = config.Registry.String()

	s.RLock()
	var endpoints []*registry.Endpoint
	for _, e := range s.handlers {
		// Only advertise non internal handlers
		if !e.Options().Internal {
			endpoints = append(endpoints, e.Endpoints()...)
		}
	}
	for e, _ := range s.subscribers {
		// Only advertise non internal subscribers
		if !e.Options().Internal {
			endpoints = append(endpoints, e.Endpoints()...)
		}
	}
	s.RUnlock()

	service := &registry.Service{
		Name:      config.Name,
		Version:   config.Version,
		Nodes:     []*registry.Node{node},
		Endpoints: endpoints,
	}

	log.Infof("Registering node: %s", node.Id)
	// create registry options
	rOpts := []registry.RegisterOption{registry.RegisterTTL(config.RegisterTTL)}

	if err := config.Registry.Register(service, rOpts...); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	for sb, _ := range s.subscribers {
		handler := s.createSubHandler(sb, s.opts)
		var opts []broker.SubscribeOption
		if queue := sb.Options().Queue; len(queue) > 0 {
			opts = append(opts, broker.QueueName(queue))
		}
		sub, err := config.Broker.Subscribe(sb.Topic(), handler, opts...)
		if err != nil {
			return err
		}
		s.subscribers[sb] = []broker.Subscriber{sub}
	}

	return nil
}

func (s *rpcServer) Deregister() error {
	config := s.Options()
	var advt, host string
	var port int

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(config.Advertise) > 0 {
		advt = config.Advertise
	} else {
		advt = config.Address
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
		Id:      config.Name + "-" + config.Id,
		Address: addr,
		Port:    port,
	}

	service := &registry.Service{
		Name:    config.Name,
		Version: config.Version,
		Nodes:   []*registry.Node{node},
	}

	log.Infof("Deregistering node: %s", node.Id)
	if err := config.Registry.Deregister(service); err != nil {
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
	registerDebugHandler(s)
	config := s.Options()

	ts, err := config.Transport.Listen(config.Address)
	if err != nil {
		return err
	}

	log.Infof("Listening on %s", ts.Addr())
	s.Lock()
	s.opts.Address = ts.Addr()
	s.Unlock()

	go ts.Accept(s.accept)

	go func() {
		ch := <-s.exit
		ch <- ts.Close()
		config.Broker.Disconnect()
	}()

	// TODO: subscribe to cruft
	return config.Broker.Connect()
}

func (s *rpcServer) Stop() error {
	ch := make(chan error)
	s.exit <- ch
	return <-ch
}

func (s *rpcServer) String() string {
	return "rpc"
}
