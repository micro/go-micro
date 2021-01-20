// Package grpc is a point to point grpc broker
package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/asim/go-micro/v3/broker"
	"github.com/asim/go-micro/v3/cmd"
	merr "github.com/asim/go-micro/v3/errors"
	log "github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/registry/cache"
	maddr "github.com/asim/go-micro/v3/util/addr"
	mnet "github.com/asim/go-micro/v3/util/net"
	mls "github.com/asim/go-micro/v3/util/tls"
	proto "github.com/asim/go-micro/plugins/broker/grpc/v3/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// grpcBroker is a point to point async broker
type grpcBroker struct {
	id      string
	address string
	opts    broker.Options
	srv     *grpc.Server
	r       registry.Registry

	sync.RWMutex
	subscribers map[string][]*grpcSubscriber
	running     bool
	exit        chan chan error
}

type grpcHandler struct {
	g *grpcBroker
}

type grpcSubscriber struct {
	opts  broker.SubscribeOptions
	id    string
	topic string
	fn    broker.Handler
	svc   *registry.Service
	hb    *grpcBroker
}

type grpcEvent struct {
	m   *broker.Message
	t   string
	err error
}

var (
	registryKey = "github.com/asim/go-micro/v3/registry"

	broadcastVersion = "ff.grpc.broadcast"
	registerTTL      = time.Minute
	registerInterval = time.Second * 30
)

func init() {
	rand.Seed(time.Now().Unix())

	cmd.DefaultBrokers["grpc"] = NewBroker
}

func newConfig(config *tls.Config) *tls.Config {
	if config == nil {
		return &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	return config
}

func newGRPCBroker(opts ...broker.Option) broker.Broker {
	options := broker.Options{
		Context: context.TODO(),
	}

	for _, o := range opts {
		o(&options)
	}

	// set address
	addr := ":0"
	if len(options.Addrs) > 0 && len(options.Addrs[0]) > 0 {
		addr = options.Addrs[0]
	}

	// get registry
	reg, ok := options.Context.Value(registryKey).(registry.Registry)
	if !ok {
		reg = registry.DefaultRegistry
	}

	h := &grpcBroker{
		id:          "grpc-broker-" + uuid.New().String(),
		address:     addr,
		opts:        options,
		r:           reg,
		srv:         grpc.NewServer(),
		subscribers: make(map[string][]*grpcSubscriber),
		exit:        make(chan chan error),
	}

	// specify the message handler
	proto.RegisterBrokerServer(h.srv, &grpcHandler{h})

	return h
}

func (h *grpcEvent) Ack() error {
	return nil
}

func (h *grpcEvent) Error() error {
	return h.err
}

func (h *grpcEvent) Message() *broker.Message {
	return h.m
}

func (h *grpcEvent) Topic() string {
	return h.t
}

func (h *grpcSubscriber) Options() broker.SubscribeOptions {
	return h.opts
}

func (h *grpcSubscriber) Topic() string {
	return h.topic
}

func (h *grpcSubscriber) Unsubscribe() error {
	return h.hb.unsubscribe(h)
}

// The grpc handler
func (h *grpcHandler) Publish(ctx context.Context, msg *proto.Message) (*proto.Empty, error) {
	if len(msg.Topic) == 0 {
		return nil, merr.InternalServerError("go.micro.broker", "Topic not found")
	}

	m := &broker.Message{
		Header: msg.Header,
		Body:   msg.Body,
	}

	p := &grpcEvent{m: m, t: msg.Topic}

	h.g.RLock()
	for _, subscriber := range h.g.subscribers[msg.Topic] {
		if msg.Id == subscriber.id {
			// sub is sync; crufty rate limiting
			// so we don't hose the cpu
			p.err = subscriber.fn(p)
		}
	}
	h.g.RUnlock()
	return new(proto.Empty), nil
}

func (h *grpcBroker) subscribe(s *grpcSubscriber) error {
	h.Lock()
	defer h.Unlock()

	if err := h.r.Register(s.svc, registry.RegisterTTL(registerTTL)); err != nil {
		return err
	}

	h.subscribers[s.topic] = append(h.subscribers[s.topic], s)
	return nil
}

func (h *grpcBroker) unsubscribe(s *grpcSubscriber) error {
	h.Lock()
	defer h.Unlock()

	var subscribers []*grpcSubscriber

	// look for subscriber
	for _, sub := range h.subscribers[s.topic] {
		// deregister and skip forward
		if sub.id == s.id {
			_ = h.r.Deregister(sub.svc)
			continue
		}
		// keep subscriber
		subscribers = append(subscribers, sub)
	}

	// set subscribers
	h.subscribers[s.topic] = subscribers

	return nil
}

func (h *grpcBroker) run(l net.Listener) {
	t := time.NewTicker(registerInterval)
	defer t.Stop()

	for {
		select {
		// heartbeat for each subscriber
		case <-t.C:
			h.RLock()
			for _, subs := range h.subscribers {
				for _, sub := range subs {
					_ = h.r.Register(sub.svc, registry.RegisterTTL(registerTTL))
				}
			}
			h.RUnlock()
		// received exit signal
		case ch := <-h.exit:
			ch <- l.Close()
			h.RLock()
			for _, subs := range h.subscribers {
				for _, sub := range subs {
					_ = h.r.Deregister(sub.svc)
				}
			}
			h.RUnlock()
			return
		}
	}
}

func (h *grpcBroker) Address() string {
	h.RLock()
	defer h.RUnlock()
	return h.address
}

func (h *grpcBroker) Connect() error {
	h.RLock()
	if h.running {
		h.RUnlock()
		return nil
	}
	h.RUnlock()

	h.Lock()
	defer h.Unlock()

	var l net.Listener
	var err error

	if h.opts.Secure || h.opts.TLSConfig != nil {
		config := h.opts.TLSConfig

		fn := func(addr string) (net.Listener, error) {
			if config == nil {
				hosts := []string{addr}

				// check if its a valid host:port
				if host, _, err := net.SplitHostPort(addr); err == nil {
					if len(host) == 0 {
						hosts = maddr.IPs()
					} else {
						hosts = []string{host}
					}
				}

				// generate a certificate
				cert, err := mls.Certificate(hosts...)
				if err != nil {
					return nil, err
				}
				config = &tls.Config{Certificates: []tls.Certificate{cert}}
			}
			return tls.Listen("tcp", addr, config)
		}

		l, err = mnet.Listen(h.address, fn)
	} else {
		fn := func(addr string) (net.Listener, error) {
			return net.Listen("tcp", addr)
		}

		l, err = mnet.Listen(h.address, fn)
	}

	if err != nil {
		return err
	}

	log.Infof("[grpc] Broker Listening on %s", l.Addr().String())
	addr := h.address
	h.address = l.Addr().String()

	go h.srv.Serve(l)

	go func() {
		h.run(l)
		h.Lock()
		h.address = addr
		h.Unlock()
	}()

	// get registry
	reg, ok := h.opts.Context.Value(registryKey).(registry.Registry)
	if !ok {
		reg = registry.DefaultRegistry
	}
	// set cache
	h.r = cache.New(reg)

	// set running
	h.running = true
	return nil
}

func (h *grpcBroker) Disconnect() error {
	h.RLock()
	if !h.running {
		h.RUnlock()
		return nil
	}
	h.RUnlock()

	h.Lock()
	defer h.Unlock()

	// stop cache
	rc, ok := h.r.(cache.Cache)
	if ok {
		rc.Stop()
	}

	// exit and return err
	ch := make(chan error)
	h.exit <- ch
	err := <-ch

	// set not running
	h.running = false
	return err
}

func (h *grpcBroker) Init(opts ...broker.Option) error {
	h.RLock()
	if h.running {
		h.RUnlock()
		return errors.New("cannot init while connected")
	}
	h.RUnlock()

	h.Lock()
	defer h.Unlock()

	for _, o := range opts {
		o(&h.opts)
	}

	if len(h.opts.Addrs) > 0 && len(h.opts.Addrs[0]) > 0 {
		h.address = h.opts.Addrs[0]
	}

	if len(h.id) == 0 {
		h.id = "broker-" + uuid.New().String()
	}

	// get registry
	reg, ok := h.opts.Context.Value(registryKey).(registry.Registry)
	if !ok {
		reg = registry.DefaultRegistry
	}

	// get cache
	if rc, ok := h.r.(cache.Cache); ok {
		rc.Stop()
	}

	// set registry
	h.r = cache.New(reg)

	return nil
}

func (h *grpcBroker) Options() broker.Options {
	return h.opts
}

func (h *grpcBroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	h.RLock()
	s, err := h.r.GetService("topic:" + topic)
	if err != nil {
		h.RUnlock()
		return err
	}
	h.RUnlock()

	m := &proto.Message{
		Topic:  topic,
		Header: make(map[string]string),
		Body:   msg.Body,
	}

	for k, v := range msg.Header {
		m.Header[k] = v
	}

	pub := func(node *registry.Node, b *proto.Message) {
		// get tls config
		config := newConfig(h.opts.TLSConfig)
		var opts []grpc.DialOption

		// check if secure is added in metadata
		if node.Metadata["secure"] == "true" {
			opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(config)))
		} else {
			opts = append(opts, grpc.WithInsecure())
		}

		m := &proto.Message{
			Topic:  b.Topic,
			Id:     node.Id,
			Header: b.Header,
			Body:   b.Body,
		}

		// dial grpc connection
		c, err := grpc.Dial(node.Address, opts...)
		if err != nil {
			log.Errorf(err.Error())
			return
		}

		defer func() {
			if err := c.Close(); err != nil {
				log.Errorf(err.Error())
				return
			}
		}()

		// publish message
		_, err = proto.NewBrokerClient(c).Publish(context.TODO(), m)
		if err != nil {
			log.Errorf(err.Error())
			return
		}
	}

	for _, service := range s {
		// only process if we have nodes
		if len(service.Nodes) == 0 {
			continue
		}

		switch service.Version {
		// broadcast version means broadcast to all nodes
		case broadcastVersion:
			for _, node := range service.Nodes {
				// publish async
				go pub(node, m)
			}
		default:
			// select node to publish to
			node := service.Nodes[rand.Int()%len(service.Nodes)]

			// publish async
			go pub(node, m)
		}
	}

	return nil
}

func (h *grpcBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	options := broker.NewSubscribeOptions(opts...)

	// parse address for host, port
	parts := strings.Split(h.Address(), ":")
	host := strings.Join(parts[:len(parts)-1], ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	addr, err := maddr.Extract(host)
	if err != nil {
		return nil, err
	}

	// create unique id
	id := h.id + "." + uuid.New().String()

	var secure bool

	if h.opts.Secure || h.opts.TLSConfig != nil {
		secure = true
	}

	// register service
	node := &registry.Node{
		Id:      id,
		Address: fmt.Sprintf("%s:%d", addr, port),
		Metadata: map[string]string{
			"secure": fmt.Sprintf("%t", secure),
		},
	}

	// check for queue group or broadcast queue
	version := options.Queue
	if len(version) == 0 {
		version = broadcastVersion
	}

	service := &registry.Service{
		Name:    "topic:" + topic,
		Version: version,
		Nodes:   []*registry.Node{node},
	}

	// generate subscriber
	subscriber := &grpcSubscriber{
		opts:  options,
		hb:    h,
		id:    id,
		topic: topic,
		fn:    handler,
		svc:   service,
	}

	// subscribe now
	if err := h.subscribe(subscriber); err != nil {
		return nil, err
	}

	// return the subscriber
	return subscriber, nil
}

func (h *grpcBroker) String() string {
	return "grpc"
}

// NewBroker returns a new grpc broker
func NewBroker(opts ...broker.Option) broker.Broker {
	return newGRPCBroker(opts...)
}
