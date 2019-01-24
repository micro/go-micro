package server

import (
	"context"
	"fmt"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/micro/go-log"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/transport"

	"github.com/micro/util/go/lib/addr"
)

type rpcServer struct {
	router *router
	exit   chan chan error

	sync.RWMutex
	opts        Options
	handlers    map[string]Handler
	subscribers map[*subscriber][]broker.Subscriber
	// used for first registration
	registered bool
	// graceful exit
	wg sync.WaitGroup
}

func newRpcServer(opts ...Option) Server {
	options := newOptions(opts...)
	return &rpcServer{
		opts:        options,
		router:      DefaultRouter,
		handlers:    make(map[string]Handler),
		subscribers: make(map[*subscriber][]broker.Subscriber),
		exit:        make(chan chan error),
	}
}

// ServeConn serves a single connection
func (s *rpcServer) ServeConn(sock transport.Socket) {
	defer func() {
		// close socket
		sock.Close()

		if r := recover(); r != nil {
			log.Log("panic recovered: ", r)
			log.Log(string(debug.Stack()))
		}
	}()

	for {
		var msg transport.Message
		if err := sock.Recv(&msg); err != nil {
			return
		}

		// add to wait group
		s.wg.Add(1)

		// we use this Timeout header to set a server deadline
		to := msg.Header["Timeout"]
		// we use this Content-Type header to identify the codec needed
		ct := msg.Header["Content-Type"]

		// strip our headers
		hdr := make(map[string]string)
		for k, v := range msg.Header {
			hdr[k] = v
		}

		// set local/remote ips
		hdr["Local"] = sock.Local()
		hdr["Remote"] = sock.Remote()

		// create new context
		ctx := metadata.NewContext(context.Background(), hdr)

		// set the timeout if we have it
		if len(to) > 0 {
			if n, err := strconv.ParseUint(to, 10, 64); err == nil {
				ctx, _ = context.WithTimeout(ctx, time.Duration(n))
			}
		}

		// no content type
		if len(ct) == 0 {
			msg.Header["Content-Type"] = DefaultContentType
			ct = DefaultContentType
		}

		// setup old protocol
		cf := setupProtocol(&msg)

		// no old codec
		if cf == nil {
			// TODO: needs better error handling
			var err error
			if cf, err = s.newCodec(ct); err != nil {
				sock.Send(&transport.Message{
					Header: map[string]string{
						"Content-Type": "text/plain",
					},
					Body: []byte(err.Error()),
				})
				s.wg.Done()
				return
			}
		}

		rcodec := newRpcCodec(&msg, sock, cf)

		// internal request
		request := &rpcRequest{
			service:     getHeader("Micro-Service", msg.Header),
			method:      getHeader("Micro-Method", msg.Header),
			endpoint:    getHeader("Micro-Endpoint", msg.Header),
			contentType: ct,
			codec:       rcodec,
			header:      msg.Header,
			body:        msg.Body,
			socket:      sock,
			stream:      true,
		}

		// internal response
		response := &rpcResponse{
			header: make(map[string]string),
			socket: sock,
			codec:  rcodec,
		}

		// set router
		r := s.opts.Router

		// if nil use default router
		if s.opts.Router == nil {
			r = s.router
		}

		// create a wrapped function
		handler := func(ctx context.Context, req Request, rsp interface{}) error {
			return r.ServeRequest(ctx, req, rsp.(Response))
		}

		for i := len(s.opts.HdlrWrappers); i > 0; i-- {
			handler = s.opts.HdlrWrappers[i-1](handler)
		}

		// TODO: handle error better
		if err := handler(ctx, request, response); err != nil {
			// write an error response
			err = rcodec.Write(&codec.Message{
				Header: msg.Header,
				Error:  err.Error(),
				Type:   codec.Error,
			}, nil)
			// could not write the error response
			if err != nil {
				log.Logf("rpc: unable to write error response: %v", err)
			}
			s.wg.Done()
			return
		}

		// done
		s.wg.Done()
	}
}

func (s *rpcServer) newCodec(contentType string) (codec.NewCodec, error) {
	if cf, ok := s.opts.Codecs[contentType]; ok {
		return cf, nil
	}
	if cf, ok := DefaultCodecs[contentType]; ok {
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
	return s.router.NewHandler(h, opts...)
}

func (s *rpcServer) Handle(h Handler) error {
	s.Lock()
	defer s.Unlock()

	if err := s.router.Handle(h); err != nil {
		return err
	}

	s.handlers[h.Name()] = h

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
	defer s.Unlock()
	_, ok = s.subscribers[sub]
	if ok {
		return fmt.Errorf("subscriber %v already exists", s)
	}
	s.subscribers[sub] = nil
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

	addr, err := addr.Extract(host)
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
	node.Metadata["protocol"] = "mucp"

	s.RLock()
	// Maps are ordered randomly, sort the keys for consistency
	var handlerList []string
	for n, e := range s.handlers {
		// Only advertise non internal handlers
		if !e.Options().Internal {
			handlerList = append(handlerList, n)
		}
	}
	sort.Strings(handlerList)

	var subscriberList []*subscriber
	for e := range s.subscribers {
		// Only advertise non internal subscribers
		if !e.Options().Internal {
			subscriberList = append(subscriberList, e)
		}
	}
	sort.Slice(subscriberList, func(i, j int) bool {
		return subscriberList[i].topic > subscriberList[j].topic
	})

	var endpoints []*registry.Endpoint
	for _, n := range handlerList {
		endpoints = append(endpoints, s.handlers[n].Endpoints()...)
	}
	for _, e := range subscriberList {
		endpoints = append(endpoints, e.Endpoints()...)
	}
	s.RUnlock()

	service := &registry.Service{
		Name:      config.Name,
		Version:   config.Version,
		Nodes:     []*registry.Node{node},
		Endpoints: endpoints,
	}

	s.Lock()
	registered := s.registered
	s.Unlock()

	if !registered {
		log.Logf("Registering node: %s", node.Id)
	}

	// create registry options
	rOpts := []registry.RegisterOption{registry.RegisterTTL(config.RegisterTTL)}

	if err := config.Registry.Register(service, rOpts...); err != nil {
		return err
	}

	// already registered? don't need to register subscribers
	if registered {
		return nil
	}

	s.Lock()
	defer s.Unlock()

	s.registered = true

	for sb, _ := range s.subscribers {
		handler := s.createSubHandler(sb, s.opts)
		var opts []broker.SubscribeOption
		if queue := sb.Options().Queue; len(queue) > 0 {
			opts = append(opts, broker.Queue(queue))
		}
		if cx := sb.Options().Context; cx != nil {
			opts = append(opts, broker.SubscribeContext(cx))
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

	addr, err := addr.Extract(host)
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

	log.Logf("Deregistering node: %s", node.Id)
	if err := config.Registry.Deregister(service); err != nil {
		return err
	}

	s.Lock()

	if !s.registered {
		s.Unlock()
		return nil
	}

	s.registered = false

	for sb, subs := range s.subscribers {
		for _, sub := range subs {
			log.Logf("Unsubscribing from topic: %s", sub.Topic())
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

	// start listening on the transport
	ts, err := config.Transport.Listen(config.Address)
	if err != nil {
		return err
	}

	log.Logf("Transport [%s] Listening on %s", config.Transport.String(), ts.Addr())

	// swap address
	s.Lock()
	addr := s.opts.Address
	s.opts.Address = ts.Addr()
	s.Unlock()

	// connect to the broker
	if err := config.Broker.Connect(); err != nil {
		return err
	}

	log.Logf("Broker [%s] Listening on %s", config.Broker.String(), config.Broker.Address())

	// announce self to the world
	if err := s.Register(); err != nil {
		log.Log("Server register error: ", err)
	}

	exit := make(chan bool)

	go func() {
		for {
			// listen for connections
			err := ts.Accept(s.ServeConn)

			// TODO: listen for messages
			// msg := broker.Exchange(service).Consume()

			select {
			// check if we're supposed to exit
			case <-exit:
				return
			// check the error and backoff
			default:
				if err != nil {
					log.Logf("Accept error: %v", err)
					time.Sleep(time.Second)
					continue
				}
			}

			// no error just exit
			return
		}
	}()

	go func() {
		t := new(time.Ticker)

		// only process if it exists
		if s.opts.RegisterInterval > time.Duration(0) {
			// new ticker
			t = time.NewTicker(s.opts.RegisterInterval)
		}

		// return error chan
		var ch chan error

	Loop:
		for {
			select {
			// register self on interval
			case <-t.C:
				if err := s.Register(); err != nil {
					log.Log("Server register error: ", err)
				}
			// wait for exit
			case ch = <-s.exit:
				t.Stop()
				close(exit)
				break Loop
			}
		}

		// deregister self
		if err := s.Deregister(); err != nil {
			log.Log("Server deregister error: ", err)
		}

		// wait for requests to finish
		if wait(s.opts.Context) {
			s.wg.Wait()
		}

		// close transport listener
		ch <- ts.Close()

		// disconnect the broker
		config.Broker.Disconnect()

		// swap back address
		s.Lock()
		s.opts.Address = addr
		s.Unlock()
	}()

	return nil
}

func (s *rpcServer) Stop() error {
	ch := make(chan error)
	s.exit <- ch
	return <-ch
}

func (s *rpcServer) String() string {
	return "rpc"
}
