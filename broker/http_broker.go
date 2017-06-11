package broker

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-log"
	"github.com/micro/go-micro/broker/codec/json"
	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/registry"
	maddr "github.com/micro/misc/lib/addr"
	mnet "github.com/micro/misc/lib/net"
	mls "github.com/micro/misc/lib/tls"
	"github.com/pborman/uuid"

	"golang.org/x/net/context"
)

// HTTP Broker is a placeholder for actual message brokers.
// This should not really be used in production but useful
// in developer where you want zero dependencies.

type httpBroker struct {
	id          string
	address     string
	unsubscribe chan *httpSubscriber
	opts        Options

	mux *http.ServeMux

	c *http.Client
	r registry.Registry

	sync.RWMutex
	subscribers map[string][]*httpSubscriber
	running     bool
	exit        chan chan error
}

type httpSubscriber struct {
	opts  SubscribeOptions
	id    string
	topic string
	ch    chan *httpSubscriber
	fn    Handler
	svc   *registry.Service
}

type httpPublication struct {
	m *Message
	t string
}

var (
	DefaultSubPath   = "/_sub"
	broadcastVersion = "ff.http.broadcast"
	registerTTL      = time.Minute
	registerInterval = time.Second * 30
)

func init() {
	rand.Seed(time.Now().Unix())
}

func newTransport(config *tls.Config) *http.Transport {
	if config == nil {
		config = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     config,
	}
	runtime.SetFinalizer(&t, func(tr **http.Transport) {
		(*tr).CloseIdleConnections()
	})
	return t
}

func newHttpBroker(opts ...Option) Broker {
	options := Options{
		Codec:   json.NewCodec(),
		Context: context.TODO(),
	}

	for _, o := range opts {
		o(&options)
	}

	addr := ":0"
	if len(options.Addrs) > 0 && len(options.Addrs[0]) > 0 {
		addr = options.Addrs[0]
	}

	reg, ok := options.Context.Value(registryKey).(registry.Registry)
	if !ok {
		reg = registry.DefaultRegistry
	}

	h := &httpBroker{
		id:          "broker-" + uuid.NewUUID().String(),
		address:     addr,
		opts:        options,
		r:           reg,
		c:           &http.Client{Transport: newTransport(options.TLSConfig)},
		subscribers: make(map[string][]*httpSubscriber),
		unsubscribe: make(chan *httpSubscriber),
		exit:        make(chan chan error),
		mux:         http.NewServeMux(),
	}

	h.mux.Handle(DefaultSubPath, h)
	return h
}

func (h *httpPublication) Ack() error {
	return nil
}

func (h *httpPublication) Message() *Message {
	return h.m
}

func (h *httpPublication) Topic() string {
	return h.t
}

func (h *httpSubscriber) Options() SubscribeOptions {
	return h.opts
}

func (h *httpSubscriber) Topic() string {
	return h.topic
}

func (h *httpSubscriber) Unsubscribe() error {
	h.ch <- h
	// artificial delay
	time.Sleep(time.Millisecond * 10)
	return nil
}

func (h *httpBroker) run(l net.Listener) {
	t := time.NewTicker(registerInterval)
	defer t.Stop()

	for {
		select {
		// heartbeat for each subscriber
		case <-t.C:
			h.RLock()
			for _, subs := range h.subscribers {
				for _, sub := range subs {
					h.r.Register(sub.svc, registry.RegisterTTL(registerTTL))
				}
			}
			h.RUnlock()
		// received exit signal
		case ch := <-h.exit:
			ch <- l.Close()
			h.Lock()
			h.running = false
			h.Unlock()
			return
		// unsubscribe subscriber
		case subscriber := <-h.unsubscribe:
			h.Lock()
			var subscribers []*httpSubscriber
			for _, sub := range h.subscribers[subscriber.topic] {
				// deregister and skip forward
				if sub.id == subscriber.id {
					h.r.Deregister(sub.svc)
					continue
				}
				subscribers = append(subscribers, sub)
			}
			h.subscribers[subscriber.topic] = subscribers
			h.Unlock()
		}
	}
}

func (h *httpBroker) start() error {
	h.Lock()
	defer h.Unlock()

	if h.running {
		return nil
	}

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

	log.Logf("Broker Listening on %s", l.Addr().String())
	h.address = l.Addr().String()

	go http.Serve(l, h.mux)
	go h.run(l)

	h.running = true
	return nil
}

func (h *httpBroker) stop() error {
	h.Lock()
	defer h.Unlock()

	if !h.running {
		return nil
	}

	ch := make(chan error)
	h.exit <- ch
	err := <-ch
	h.running = false
	return err
}

func (h *httpBroker) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		err := errors.BadRequest("go.micro.broker", "Method not allowed")
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
		return
	}
	defer req.Body.Close()

	req.ParseForm()

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		errr := errors.InternalServerError("go.micro.broker", "Error reading request body: %v", err)
		w.WriteHeader(500)
		w.Write([]byte(errr.Error()))
		return
	}

	var m *Message
	if err = h.opts.Codec.Unmarshal(b, &m); err != nil {
		errr := errors.InternalServerError("go.micro.broker", "Error parsing request body: %v", err)
		w.WriteHeader(500)
		w.Write([]byte(errr.Error()))
		return
	}

	topic := m.Header[":topic"]
	delete(m.Header, ":topic")

	if len(topic) == 0 {
		errr := errors.InternalServerError("go.micro.broker", "Topic not found")
		w.WriteHeader(500)
		w.Write([]byte(errr.Error()))
		return
	}

	p := &httpPublication{m: m, t: topic}
	id := req.Form.Get("id")

	h.RLock()
	for _, subscriber := range h.subscribers[topic] {
		if id == subscriber.id {
			// sub is sync; crufty rate limiting
			// so we don't hose the cpu
			subscriber.fn(p)
		}
	}
	h.RUnlock()
}

func (h *httpBroker) Address() string {
	return h.address
}

func (h *httpBroker) Connect() error {
	return h.start()
}

func (h *httpBroker) Disconnect() error {
	return h.stop()
}

func (h *httpBroker) Init(opts ...Option) error {
	for _, o := range opts {
		o(&h.opts)
	}

	if len(h.id) == 0 {
		h.id = "broker-" + uuid.NewUUID().String()
	}

	reg, ok := h.opts.Context.Value(registryKey).(registry.Registry)
	if !ok {
		reg = registry.DefaultRegistry
	}

	h.r = reg

	return nil
}

func (h *httpBroker) Options() Options {
	return h.opts
}

func (h *httpBroker) Publish(topic string, msg *Message, opts ...PublishOption) error {
	s, err := h.r.GetService("topic:" + topic)
	if err != nil {
		return err
	}

	m := &Message{
		Header: make(map[string]string),
		Body:   msg.Body,
	}

	for k, v := range msg.Header {
		m.Header[k] = v
	}

	m.Header[":topic"] = topic

	b, err := h.opts.Codec.Marshal(m)
	if err != nil {
		return err
	}

	fn := func(node *registry.Node, b []byte) {
		scheme := "http"

		// check if secure is added in metadata
		if node.Metadata["secure"] == "true" {
			scheme = "https"
		}

		vals := url.Values{}
		vals.Add("id", node.Id)

		uri := fmt.Sprintf("%s://%s:%d%s?%s", scheme, node.Address, node.Port, DefaultSubPath, vals.Encode())
		r, err := h.c.Post(uri, "application/json", bytes.NewReader(b))
		if err == nil {
			io.Copy(ioutil.Discard, r.Body)
			r.Body.Close()
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
				go fn(node, b)
			}

		default:
			// select node to publish to
			node := service.Nodes[rand.Int()%len(service.Nodes)]

			// publish async
			go fn(node, b)
		}
	}

	return nil
}

func (h *httpBroker) Subscribe(topic string, handler Handler, opts ...SubscribeOption) (Subscriber, error) {
	opt := newSubscribeOptions(opts...)

	// parse address for host, port
	parts := strings.Split(h.Address(), ":")
	host := strings.Join(parts[:len(parts)-1], ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	addr, err := maddr.Extract(host)
	if err != nil {
		return nil, err
	}

	id := uuid.NewUUID().String()

	var secure bool

	if h.opts.Secure || h.opts.TLSConfig != nil {
		secure = true
	}

	// register service
	node := &registry.Node{
		Id:      h.id + "." + id,
		Address: addr,
		Port:    port,
		Metadata: map[string]string{
			"secure": fmt.Sprintf("%t", secure),
		},
	}

	version := opt.Queue
	if len(version) == 0 {
		version = broadcastVersion
	}

	service := &registry.Service{
		Name:    "topic:" + topic,
		Version: version,
		Nodes:   []*registry.Node{node},
	}

	subscriber := &httpSubscriber{
		opts:  opt,
		id:    h.id + "." + id,
		topic: topic,
		ch:    h.unsubscribe,
		fn:    handler,
		svc:   service,
	}

	if err := h.r.Register(service, registry.RegisterTTL(registerTTL)); err != nil {
		return nil, err
	}

	h.Lock()
	h.subscribers[topic] = append(h.subscribers[topic], subscriber)
	h.Unlock()
	return subscriber, nil
}

func (h *httpBroker) String() string {
	return "http"
}
