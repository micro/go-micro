package server

import (
	"strconv"
	"strings"
	"sync"

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
	opts     options
	handlers map[string]Handler
}

func newRpcServer(opts ...Option) Server {
	return &rpcServer{
		opts:     newOptions(opts...),
		rpc:      rpc.NewServer(),
		handlers: make(map[string]Handler),
		exit:     make(chan chan error),
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

func (s *rpcServer) Register() error {
	// parse address for host, port
	config := s.Config()
	var host string
	var port int
	parts := strings.Split(config.Address(), ":")
	if len(parts) > 1 {
		host = strings.Join(parts[:len(parts)-1], ":")
		port, _ = strconv.Atoi(parts[len(parts)-1])
	} else {
		host = parts[0]
	}

	// register service
	node := &registry.Node{
		Id:       config.Id(),
		Address:  host,
		Port:     port,
		Metadata: config.Metadata(),
	}

	s.RLock()
	var endpoints []*registry.Endpoint
	for _, e := range s.handlers {
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
	return config.registry.Register(service)
}

func (s *rpcServer) Deregister() error {
	config := s.Config()
	var host string
	var port int
	parts := strings.Split(config.Address(), ":")
	if len(parts) > 1 {
		host = strings.Join(parts[:len(parts)-1], ":")
		port, _ = strconv.Atoi(parts[len(parts)-1])
	} else {
		host = parts[0]
	}

	node := &registry.Node{
		Id:      config.Id(),
		Address: host,
		Port:    port,
	}

	service := &registry.Service{
		Name:    config.Name(),
		Version: config.Version(),
		Nodes:   []*registry.Node{node},
	}

	return config.registry.Deregister(service)
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
	}()

	return nil
}

func (s *rpcServer) Stop() error {
	ch := make(chan error)
	s.exit <- ch
	return <-ch
}
