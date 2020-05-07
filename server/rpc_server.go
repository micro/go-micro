package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/broker"
	"github.com/micro/go-micro/v2/codec"
	raw "github.com/micro/go-micro/v2/codec/bytes"
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/metadata"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/transport"
	"github.com/micro/go-micro/v2/util/addr"
	mnet "github.com/micro/go-micro/v2/util/net"
	"github.com/micro/go-micro/v2/util/socket"
)

type rpcServer struct {
	router *router
	exit   chan chan error

	sync.RWMutex
	opts        Options
	handlers    map[string]Handler
	subscribers map[Subscriber][]broker.Subscriber
	// marks the serve as started
	started bool
	// used for first registration
	registered bool
	// subscribe to service name
	subscriber broker.Subscriber
	// graceful exit
	wg *sync.WaitGroup

	rsvc *registry.Service
}

func newRpcServer(opts ...Option) Server {
	options := newOptions(opts...)
	router := newRpcRouter()
	router.hdlrWrappers = options.HdlrWrappers
	router.subWrappers = options.SubWrappers

	return &rpcServer{
		opts:        options,
		router:      router,
		handlers:    make(map[string]Handler),
		subscribers: make(map[Subscriber][]broker.Subscriber),
		exit:        make(chan chan error),
		wg:          wait(options.Context),
	}
}

// HandleEvent handles inbound messages to the service directly
// TODO: handle requests from an event. We won't send a response.
func (s *rpcServer) HandleEvent(e broker.Event) error {
	// formatting horrible cruft
	msg := e.Message()

	if msg.Header == nil {
		// create empty map in case of headers empty to avoid panic later
		msg.Header = make(map[string]string)
	}

	// get codec
	ct := msg.Header["Content-Type"]

	// default content type
	if len(ct) == 0 {
		msg.Header["Content-Type"] = DefaultContentType
		ct = DefaultContentType
	}

	// get codec
	cf, err := s.newCodec(ct)
	if err != nil {
		return err
	}

	// copy headers
	hdr := make(map[string]string, len(msg.Header))
	for k, v := range msg.Header {
		hdr[k] = v
	}

	// create context
	ctx := metadata.NewContext(context.Background(), hdr)

	// TODO: inspect message header
	// Micro-Service means a request
	// Micro-Topic means a message

	rpcMsg := &rpcMessage{
		topic:       msg.Header["Micro-Topic"],
		contentType: ct,
		payload:     &raw.Frame{Data: msg.Body},
		codec:       cf,
		header:      msg.Header,
		body:        msg.Body,
	}

	// existing router
	r := Router(s.router)

	// if the router is present then execute it
	if s.opts.Router != nil {
		// create a wrapped function
		handler := s.opts.Router.ProcessMessage

		// execute the wrapper for it
		for i := len(s.opts.SubWrappers); i > 0; i-- {
			handler = s.opts.SubWrappers[i-1](handler)
		}

		// set the router
		r = rpcRouter{m: handler}
	}

	return r.ProcessMessage(ctx, rpcMsg)
}

// ServeConn serves a single connection
func (s *rpcServer) ServeConn(sock transport.Socket) {
	// global error tracking
	var gerr error
	// streams are multiplexed on Micro-Stream or Micro-Id header
	pool := socket.NewPool()

	// get global waitgroup
	s.Lock()
	gg := s.wg
	s.Unlock()

	// waitgroup to wait for processing to finish
	wg := &waitGroup{
		gg: gg,
	}

	defer func() {
		// only wait if there's no error
		if gerr == nil {
			// wait till done
			wg.Wait()
		}

		// close all the sockets for this connection
		pool.Close()

		// close underlying socket
		sock.Close()

		// recover any panics
		if r := recover(); r != nil {
			if logger.V(logger.ErrorLevel, log) {
				log.Error("panic recovered: ", r)
				log.Error(string(debug.Stack()))
			}
		}
	}()

	for {
		var msg transport.Message
		// process inbound messages one at a time
		if err := sock.Recv(&msg); err != nil {
			// set a global error and return
			// we're saying we essentially can't
			// use the socket anymore
			gerr = err
			return
		}

		// check the message header for
		// Micro-Service is a request
		// Micro-Topic is a message
		if t := msg.Header["Micro-Topic"]; len(t) > 0 {
			// process the event
			ev := newEvent(msg)
			// TODO: handle the error event
			if err := s.HandleEvent(ev); err != nil {
				msg.Header["Micro-Error"] = err.Error()
			}
			// write back some 200
			if err := sock.Send(&transport.Message{
				Header: msg.Header,
			}); err != nil {
				gerr = err
				break
			}
			// we're done
			continue
		}

		// business as usual

		// use Micro-Stream as the stream identifier
		// in the event its blank we'll always process
		// on the same socket
		id := msg.Header["Micro-Stream"]

		// if there's no stream id then its a standard request
		// use the Micro-Id
		if len(id) == 0 {
			id = msg.Header["Micro-Id"]
		}

		// check stream id
		var stream bool

		if v := getHeader("Micro-Stream", msg.Header); len(v) > 0 {
			stream = true
		}

		// check if we have an existing socket
		psock, ok := pool.Get(id)

		// if we don't have a socket and its a stream
		if !ok && stream {
			// check if its a last stream EOS error
			err := msg.Header["Micro-Error"]
			if err == lastStreamResponseError.Error() {
				pool.Release(psock)
				continue
			}
		}

		// got an existing socket already
		if ok {
			// we're starting processing
			wg.Add(1)

			// pass the message to that existing socket
			if err := psock.Accept(&msg); err != nil {
				// release the socket if there's an error
				pool.Release(psock)
			}

			// done waiting
			wg.Done()

			// continue to the next message
			continue
		}

		// no socket was found so its new
		// set the local and remote values
		psock.SetLocal(sock.Local())
		psock.SetRemote(sock.Remote())

		// load the socket with the current message
		psock.Accept(&msg)

		// now walk the usual path

		// we use this Timeout header to set a server deadline
		to := msg.Header["Timeout"]
		// we use this Content-Type header to identify the codec needed
		ct := msg.Header["Content-Type"]

		// copy the message headers
		hdr := make(map[string]string, len(msg.Header))
		for k, v := range msg.Header {
			hdr[k] = v
		}

		// set local/remote ips
		hdr["Local"] = sock.Local()
		hdr["Remote"] = sock.Remote()

		// create new context with the metadata
		ctx := metadata.NewContext(context.Background(), hdr)

		// set the timeout from the header if we have it
		if len(to) > 0 {
			if n, err := strconv.ParseUint(to, 10, 64); err == nil {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Duration(n))
				defer cancel()
			}
		}

		// if there's no content type default it
		if len(ct) == 0 {
			msg.Header["Content-Type"] = DefaultContentType
			ct = DefaultContentType
		}

		// setup old protocol
		cf := setupProtocol(&msg)

		// no legacy codec needed
		if cf == nil {
			var err error
			// try get a new codec
			if cf, err = s.newCodec(ct); err != nil {
				// no codec found so send back an error
				if err := sock.Send(&transport.Message{
					Header: map[string]string{
						"Content-Type": "text/plain",
					},
					Body: []byte(err.Error()),
				}); err != nil {
					gerr = err
				}

				// release the socket we just created
				pool.Release(psock)
				// now continue
				continue
			}
		}

		// create a new rpc codec based on the pseudo socket and codec
		rcodec := newRpcCodec(&msg, psock, cf)
		// check the protocol as well
		protocol := rcodec.String()

		// internal request
		request := &rpcRequest{
			service:     getHeader("Micro-Service", msg.Header),
			method:      getHeader("Micro-Method", msg.Header),
			endpoint:    getHeader("Micro-Endpoint", msg.Header),
			contentType: ct,
			codec:       rcodec,
			header:      msg.Header,
			body:        msg.Body,
			socket:      psock,
			stream:      stream,
		}

		// internal response
		response := &rpcResponse{
			header: make(map[string]string),
			socket: psock,
			codec:  rcodec,
		}

		// set router
		r := Router(s.router)

		// if not nil use the router specified
		if s.opts.Router != nil {
			// create a wrapped function
			handler := func(ctx context.Context, req Request, rsp interface{}) error {
				return s.opts.Router.ServeRequest(ctx, req, rsp.(Response))
			}

			// execute the wrapper for it
			for i := len(s.opts.HdlrWrappers); i > 0; i-- {
				handler = s.opts.HdlrWrappers[i-1](handler)
			}

			// set the router
			r = rpcRouter{h: handler}
		}

		// process the outbound messages from the socket
		go func(id string, psock *socket.Socket) {
			// wait for processing to exit
			wg.Add(1)

			defer func() {
				// TODO: don't hack this but if its grpc just break out of the stream
				// We do this because the underlying connection is h2 and its a stream
				switch protocol {
				case "grpc":
					sock.Close()
				}
				// release the socket
				pool.Release(psock)
				// signal we're done
				wg.Done()

				// recover any panics for outbound process
				if r := recover(); r != nil {
					if logger.V(logger.ErrorLevel, log) {
						log.Error("panic recovered: ", r)
						log.Error(string(debug.Stack()))
					}
				}
			}()

			for {
				// get the message from our internal handler/stream
				m := new(transport.Message)
				if err := psock.Process(m); err != nil {
					return
				}

				// send the message back over the socket
				if err := sock.Send(m); err != nil {
					return
				}
			}
		}(id, psock)

		// serve the request in a go routine as this may be a stream
		go func(id string, psock *socket.Socket) {
			// add to the waitgroup
			wg.Add(1)

			defer func() {
				// release the socket
				pool.Release(psock)
				// signal we're done
				wg.Done()

				// recover any panics for call handler
				if r := recover(); r != nil {
					log.Error("panic recovered: ", r)
					log.Error(string(debug.Stack()))
				}
			}()

			// serve the actual request using the request router
			if serveRequestError := r.ServeRequest(ctx, request, response); serveRequestError != nil {
				// write an error response
				writeError := rcodec.Write(&codec.Message{
					Header: msg.Header,
					Error:  serveRequestError.Error(),
					Type:   codec.Error,
				}, nil)

				// if the server request is an EOS error we let the socket know
				// sometimes the socket is already closed on the other side, so we can ignore that error
				alreadyClosed := serveRequestError == lastStreamResponseError && writeError == io.EOF

				// could not write error response
				if writeError != nil && !alreadyClosed {
					log.Debugf("rpc: unable to write error response: %v", writeError)
				}
			}
		}(id, psock)
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
	defer s.Unlock()

	for _, opt := range opts {
		opt(&s.opts)
	}
	// update router if its the default
	if s.opts.Router == nil {
		r := newRpcRouter()
		r.hdlrWrappers = s.opts.HdlrWrappers
		r.serviceMap = s.router.serviceMap
		r.subWrappers = s.opts.SubWrappers
		s.router = r
	}

	s.rsvc = nil

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
	return s.router.NewSubscriber(topic, sb, opts...)
}

func (s *rpcServer) Subscribe(sb Subscriber) error {
	s.Lock()
	defer s.Unlock()

	if err := s.router.Subscribe(sb); err != nil {
		return err
	}

	s.subscribers[sb] = nil
	return nil
}

func (s *rpcServer) Register() error {

	s.RLock()
	rsvc := s.rsvc
	config := s.Options()
	s.RUnlock()

	if rsvc != nil {
		rOpts := []registry.RegisterOption{registry.RegisterTTL(config.RegisterTTL)}
		if err := config.Registry.Register(rsvc, rOpts...); err != nil {
			return err
		}

		return nil
	}

	var err error
	var advt, host, port string
	var cacheService bool

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(config.Advertise) > 0 {
		advt = config.Advertise
	} else {
		advt = config.Address
	}

	if cnt := strings.Count(advt, ":"); cnt >= 1 {
		// ipv6 address in format [host]:port or ipv4 host:port
		host, port, err = net.SplitHostPort(advt)
		if err != nil {
			return err
		}
	} else {
		host = advt
	}

	if ip := net.ParseIP(host); ip != nil {
		cacheService = true
	}

	addr, err := addr.Extract(host)
	if err != nil {
		return err
	}

	// make copy of metadata
	md := metadata.Copy(config.Metadata)

	// mq-rpc(eg. nats) doesn't need the port. its addr is queue name.
	if port != "" {
		addr = mnet.HostPort(addr, port)
	}

	// register service
	node := &registry.Node{
		Id:       config.Name + "-" + config.Id,
		Address:  addr,
		Metadata: md,
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

	var subscriberList []Subscriber
	for e := range s.subscribers {
		// Only advertise non internal subscribers
		if !e.Options().Internal {
			subscriberList = append(subscriberList, e)
		}
	}

	sort.Slice(subscriberList, func(i, j int) bool {
		return subscriberList[i].Topic() > subscriberList[j].Topic()
	})

	endpoints := make([]*registry.Endpoint, 0, len(handlerList)+len(subscriberList))

	for _, n := range handlerList {
		endpoints = append(endpoints, s.handlers[n].Endpoints()...)
	}

	for _, e := range subscriberList {
		endpoints = append(endpoints, e.Endpoints()...)
	}

	service := &registry.Service{
		Name:      config.Name,
		Version:   config.Version,
		Nodes:     []*registry.Node{node},
		Endpoints: endpoints,
	}

	// get registered value
	registered := s.registered

	s.RUnlock()

	if !registered {
		if logger.V(logger.InfoLevel, logger.DefaultLogger) {
			log.Infof("Registry [%s] Registering node: %s", config.Registry.String(), node.Id)
		}
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

	// set what we're advertising
	s.opts.Advertise = addr

	// router can exchange messages
	if s.opts.Router != nil {
		// subscribe to the topic with own name
		sub, err := s.opts.Broker.Subscribe(config.Name, s.HandleEvent)
		if err != nil {
			return err
		}

		// save the subscriber
		s.subscriber = sub
	}

	// subscribe for all of the subscribers
	for sb := range s.subscribers {
		var opts []broker.SubscribeOption
		if queue := sb.Options().Queue; len(queue) > 0 {
			opts = append(opts, broker.Queue(queue))
		}

		if cx := sb.Options().Context; cx != nil {
			opts = append(opts, broker.SubscribeContext(cx))
		}

		if !sb.Options().AutoAck {
			opts = append(opts, broker.DisableAutoAck())
		}

		sub, err := config.Broker.Subscribe(sb.Topic(), s.HandleEvent, opts...)
		if err != nil {
			return err
		}
		if logger.V(logger.InfoLevel, logger.DefaultLogger) {
			log.Infof("Subscribing to topic: %s", sub.Topic())
		}
		s.subscribers[sb] = []broker.Subscriber{sub}
	}
	if cacheService {
		s.rsvc = service
	}
	s.registered = true

	return nil
}

func (s *rpcServer) Deregister() error {
	var err error
	var advt, host, port string

	s.RLock()
	config := s.Options()
	s.RUnlock()

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(config.Advertise) > 0 {
		advt = config.Advertise
	} else {
		advt = config.Address
	}

	if cnt := strings.Count(advt, ":"); cnt >= 1 {
		// ipv6 address in format [host]:port or ipv4 host:port
		host, port, err = net.SplitHostPort(advt)
		if err != nil {
			return err
		}
	} else {
		host = advt
	}

	addr, err := addr.Extract(host)
	if err != nil {
		return err
	}

	// mq-rpc(eg. nats) doesn't need the port. its addr is queue name.
	if port != "" {
		addr = mnet.HostPort(addr, port)
	}

	node := &registry.Node{
		Id:      config.Name + "-" + config.Id,
		Address: addr,
	}

	service := &registry.Service{
		Name:    config.Name,
		Version: config.Version,
		Nodes:   []*registry.Node{node},
	}

	if logger.V(logger.InfoLevel, logger.DefaultLogger) {
		log.Infof("Registry [%s] Deregistering node: %s", config.Registry.String(), node.Id)
	}
	if err := config.Registry.Deregister(service); err != nil {
		return err
	}

	s.Lock()
	s.rsvc = nil

	if !s.registered {
		s.Unlock()
		return nil
	}

	s.registered = false

	// close the subscriber
	if s.subscriber != nil {
		s.subscriber.Unsubscribe()
		s.subscriber = nil
	}

	for sb, subs := range s.subscribers {
		for _, sub := range subs {
			if logger.V(logger.InfoLevel, logger.DefaultLogger) {
				log.Infof("Unsubscribing %s from topic: %s", node.Id, sub.Topic())
			}
			sub.Unsubscribe()
		}
		s.subscribers[sb] = nil
	}

	s.Unlock()
	return nil
}

func (s *rpcServer) Start() error {
	s.RLock()
	if s.started {
		s.RUnlock()
		return nil
	}
	s.RUnlock()

	config := s.Options()

	// start listening on the transport
	ts, err := config.Transport.Listen(config.Address)
	if err != nil {
		return err
	}

	if logger.V(logger.InfoLevel, logger.DefaultLogger) {
		log.Infof("Transport [%s] Listening on %s", config.Transport.String(), ts.Addr())
	}

	// swap address
	s.Lock()
	addr := s.opts.Address
	s.opts.Address = ts.Addr()
	s.Unlock()

	bname := config.Broker.String()

	// connect to the broker
	if err := config.Broker.Connect(); err != nil {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			log.Errorf("Broker [%s] connect error: %v", bname, err)
		}
		return err
	}

	if logger.V(logger.InfoLevel, logger.DefaultLogger) {
		log.Infof("Broker [%s] Connected to %s", bname, config.Broker.Address())
	}

	// use RegisterCheck func before register
	if err = s.opts.RegisterCheck(s.opts.Context); err != nil {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			log.Errorf("Server %s-%s register check error: %s", config.Name, config.Id, err)
		}
	} else {
		// announce self to the world
		if err = s.Register(); err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				log.Errorf("Server %s-%s register error: %s", config.Name, config.Id, err)
			}
		}
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
					if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
						log.Errorf("Accept error: %v", err)
					}
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
				s.RLock()
				registered := s.registered
				s.RUnlock()
				rerr := s.opts.RegisterCheck(s.opts.Context)
				if rerr != nil && registered {
					if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
						log.Errorf("Server %s-%s register check error: %s, deregister it", config.Name, config.Id, err)
					}
					// deregister self in case of error
					if err := s.Deregister(); err != nil {
						if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
							log.Errorf("Server %s-%s deregister error: %s", config.Name, config.Id, err)
						}
					}
				} else if rerr != nil && !registered {
					if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
						log.Errorf("Server %s-%s register check error: %s", config.Name, config.Id, err)
					}
					continue
				}
				if err := s.Register(); err != nil {
					if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
						log.Errorf("Server %s-%s register error: %s", config.Name, config.Id, err)
					}
				}
			// wait for exit
			case ch = <-s.exit:
				t.Stop()
				close(exit)
				break Loop
			}
		}

		s.RLock()
		registered := s.registered
		s.RUnlock()
		if registered {
			// deregister self
			if err := s.Deregister(); err != nil {
				if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
					log.Errorf("Server %s-%s deregister error: %s", config.Name, config.Id, err)
				}
			}
		}

		s.Lock()
		swg := s.wg
		s.Unlock()

		// wait for requests to finish
		if swg != nil {
			swg.Wait()
		}

		// close transport listener
		ch <- ts.Close()

		if logger.V(logger.InfoLevel, logger.DefaultLogger) {
			log.Infof("Broker [%s] Disconnected from %s", bname, config.Broker.Address())
		}
		// disconnect the broker
		if err := config.Broker.Disconnect(); err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				log.Errorf("Broker [%s] Disconnect error: %v", bname, err)
			}
		}

		// swap back address
		s.Lock()
		s.opts.Address = addr
		s.Unlock()
	}()

	// mark the server as started
	s.Lock()
	s.started = true
	s.Unlock()

	return nil
}

func (s *rpcServer) Stop() error {
	s.RLock()
	if !s.started {
		s.RUnlock()
		return nil
	}
	s.RUnlock()

	ch := make(chan error)
	s.exit <- ch

	err := <-ch
	s.Lock()
	s.started = false
	s.Unlock()

	return err
}

func (s *rpcServer) String() string {
	return "mucp"
}
