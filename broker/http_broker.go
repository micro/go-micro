package broker

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
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

	"github.com/google/uuid"
	"github.com/micro/go-micro/codec/json"
	merr "github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-rcache"
	maddr "github.com/micro/util/go/lib/addr"
	mnet "github.com/micro/util/go/lib/net"
	mls "github.com/micro/util/go/lib/tls"
	"golang.org/x/net/http2"
)

// HTTP Broker is a point to point async broker
type httpBroker struct {
	id      string
	address string
	opts    Options

	mux *http.ServeMux

	c *http.Client
	r registry.Registry

	sync.RWMutex
	subscribers map[string][]*httpSubscriber
	running     bool
	exit        chan chan error

	// offline message inbox
	mtx   sync.RWMutex
	inbox map[string][][]byte
}

type httpSubscriber struct {
	opts  SubscribeOptions
	id    string
	topic string
	fn    Handler
	svc   *registry.Service
	hb    *httpBroker
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

	dialTLS := func(network string, addr string) (net.Conn, error) {
		return tls.Dial(network, addr, config)
	}

	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		DialTLS:             dialTLS,
	}
	runtime.SetFinalizer(&t, func(tr **http.Transport) {
		(*tr).CloseIdleConnections()
	})

	// setup http2
	http2.ConfigureTransport(t)

	return t
}

func newHttpBroker(opts ...Option) Broker {
	options := Options{
		Codec:   json.Marshaler{},
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

	h := &httpBroker{
		id:          "broker-" + uuid.New().String(),
		address:     addr,
		opts:        options,
		r:           reg,
		c:           &http.Client{Transport: newTransport(options.TLSConfig)},
		subscribers: make(map[string][]*httpSubscriber),
		exit:        make(chan chan error),
		mux:         http.NewServeMux(),
		inbox:       make(map[string][][]byte),
	}

	// specify the message handler
	h.mux.Handle(DefaultSubPath, h)

	// get optional handlers
	if h.opts.Context != nil {
		handlers, ok := h.opts.Context.Value("http_handlers").(map[string]http.Handler)
		if ok {
			for pattern, handler := range handlers {
				h.mux.Handle(pattern, handler)
			}
		}
	}

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
	return h.hb.unsubscribe(h)
}

func (h *httpBroker) saveMessage(topic string, msg []byte) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	// get messages
	c := h.inbox[topic]

	// save message
	c = append(c, msg)

	// max length 64
	if len(c) > 64 {
		c = c[:64]
	}

	// save inbox
	h.inbox[topic] = c
}

func (h *httpBroker) getMessage(topic string, num int) [][]byte {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	// get messages
	c, ok := h.inbox[topic]
	if !ok {
		return nil
	}

	// more message than requests
	if len(c) >= num {
		msg := c[:num]
		h.inbox[topic] = c[num:]
		return msg
	}

	// reset inbox
	h.inbox[topic] = nil

	// return all messages
	return c
}

func (h *httpBroker) subscribe(s *httpSubscriber) error {
	h.Lock()
	defer h.Unlock()

	if err := h.r.Register(s.svc, registry.RegisterTTL(registerTTL)); err != nil {
		return err
	}

	h.subscribers[s.topic] = append(h.subscribers[s.topic], s)
	return nil
}

func (h *httpBroker) unsubscribe(s *httpSubscriber) error {
	h.Lock()
	defer h.Unlock()

	var subscribers []*httpSubscriber

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

func (h *httpBroker) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		err := merr.BadRequest("go.micro.broker", "Method not allowed")
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
		return
	}
	defer req.Body.Close()

	req.ParseForm()

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		errr := merr.InternalServerError("go.micro.broker", "Error reading request body: %v", err)
		w.WriteHeader(500)
		w.Write([]byte(errr.Error()))
		return
	}

	var m *Message
	if err = h.opts.Codec.Unmarshal(b, &m); err != nil {
		errr := merr.InternalServerError("go.micro.broker", "Error parsing request body: %v", err)
		w.WriteHeader(500)
		w.Write([]byte(errr.Error()))
		return
	}

	topic := m.Header[":topic"]
	delete(m.Header, ":topic")

	if len(topic) == 0 {
		errr := merr.InternalServerError("go.micro.broker", "Topic not found")
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
	h.RLock()
	defer h.RUnlock()
	return h.address
}

func (h *httpBroker) Connect() error {
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

	addr := h.address
	h.address = l.Addr().String()

	go http.Serve(l, h.mux)
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
	// set rcache
	h.r = rcache.New(reg)

	// set running
	h.running = true
	return nil
}

func (h *httpBroker) Disconnect() error {

	h.RLock()
	if !h.running {
		h.RUnlock()
		return nil
	}
	h.RUnlock()

	h.Lock()
	defer h.Unlock()

	// stop rcache
	rc, ok := h.r.(rcache.Cache)
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

func (h *httpBroker) Init(opts ...Option) error {
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

	// get rcache
	if rc, ok := h.r.(rcache.Cache); ok {
		rc.Stop()
	}

	// set registry
	h.r = rcache.New(reg)

	// reconfigure tls config
	if c := h.opts.TLSConfig; c != nil {
		h.c = &http.Client{
			Transport: newTransport(c),
		}
	}

	return nil
}

func (h *httpBroker) Options() Options {
	return h.opts
}

func (h *httpBroker) Publish(topic string, msg *Message, opts ...PublishOption) error {
	// create the message first
	m := &Message{
		Header: make(map[string]string),
		Body:   msg.Body,
	}

	for k, v := range msg.Header {
		m.Header[k] = v
	}

	m.Header[":topic"] = topic

	// encode the message
	b, err := h.opts.Codec.Marshal(m)
	if err != nil {
		return err
	}

	// save the message
	h.saveMessage(topic, b)

	// now attempt to get the service
	h.RLock()
	s, err := h.r.GetService("topic:" + topic)
	if err != nil {
		h.RUnlock()
		// ignore error
		return nil
	}
	h.RUnlock()

	pub := func(node *registry.Node, t string, b []byte) error {
		scheme := "http"

		// check if secure is added in metadata
		if node.Metadata["secure"] == "true" {
			scheme = "https"
		}

		vals := url.Values{}
		vals.Add("id", node.Id)

		uri := fmt.Sprintf("%s://%s:%d%s?%s", scheme, node.Address, node.Port, DefaultSubPath, vals.Encode())
		r, err := h.c.Post(uri, "application/json", bytes.NewReader(b))
		if err != nil {
			return err
		}

		// discard response body
		io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()
		return nil
	}

	srv := func(s []*registry.Service, b []byte) {
		for _, service := range s {
			// only process if we have nodes
			if len(service.Nodes) == 0 {
				continue
			}

			switch service.Version {
			// broadcast version means broadcast to all nodes
			case broadcastVersion:
				var success bool

				// publish to all nodes
				for _, node := range service.Nodes {
					// publish async
					if err := pub(node, topic, b); err == nil {
						success = true
					}
				}

				// save if it failed to publish at least once
				if !success {
					h.saveMessage(topic, b)
				}
			default:
				// select node to publish to
				node := service.Nodes[rand.Int()%len(service.Nodes)]

				// publish async to one node
				if err := pub(node, topic, b); err != nil {
					// if failed save it
					h.saveMessage(topic, b)
				}
			}
		}
	}

	// do the rest async
	go func() {
		// get a third of the backlog
		messages := h.getMessage(topic, 8)
		delay := (len(messages) > 1)

		// publish all the messages
		for _, msg := range messages {
			// serialize here
			srv(s, msg)

			// sending a backlog of messages
			if delay {
				time.Sleep(time.Millisecond * 100)
			}
		}
	}()

	return nil
}

func (h *httpBroker) Subscribe(topic string, handler Handler, opts ...SubscribeOption) (Subscriber, error) {
	options := NewSubscribeOptions(opts...)

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
		Address: addr,
		Port:    port,
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
	subscriber := &httpSubscriber{
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

func (h *httpBroker) String() string {
	return "http"
}
