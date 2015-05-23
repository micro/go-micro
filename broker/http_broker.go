package broker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"code.google.com/p/go-uuid/uuid"
	log "github.com/golang/glog"
	c "github.com/myodc/go-micro/context"
	"github.com/myodc/go-micro/errors"
	"github.com/myodc/go-micro/registry"

	"golang.org/x/net/context"
)

type httpBroker struct {
	id          string
	address     string
	unsubscribe chan *httpSubscriber

	sync.RWMutex
	subscribers map[string][]*httpSubscriber
	running     bool
	exit        chan chan error
}

type httpSubscriber struct {
	id    string
	topic string
	ch    chan *httpSubscriber
	fn    func(context.Context, *Message)
	svc   registry.Service
}

// used in brokers where there is no support for headers
type envelope struct {
	Header  map[string]string
	Message *Message
}

var (
	DefaultSubPath = "/_sub"
)

func newHttpBroker(addrs []string, opt ...Option) Broker {
	addr := ":0"
	if len(addrs) > 0 {
		addr = addrs[0]
	}

	return &httpBroker{
		id:          Id,
		address:     addr,
		subscribers: make(map[string][]*httpSubscriber),
		unsubscribe: make(chan *httpSubscriber),
		exit:        make(chan chan error),
	}
}

func (h *httpSubscriber) Topic() string {
	return h.topic
}

func (h *httpSubscriber) Unsubscribe() error {
	h.ch <- h
	return nil
}

func (h *httpBroker) start() error {
	h.Lock()
	defer h.Unlock()

	if h.running {
		return nil
	}

	l, err := net.Listen("tcp", h.address)
	if err != nil {
		return err
	}

	log.Infof("Broker Listening on %s", l.Addr().String())
	h.address = l.Addr().String()

	go http.Serve(l, h)

	go func() {
		ce := make(chan os.Signal, 1)
		signal.Notify(ce, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)

		for {
			select {
			case ch := <-h.exit:
				ch <- l.Close()
				h.Lock()
				h.running = false
				h.Unlock()
				return
			case <-ce:
				h.stop()
			case subscriber := <-h.unsubscribe:
				h.Lock()
				var subscribers []*httpSubscriber
				for _, sub := range h.subscribers[subscriber.topic] {
					if sub.id == subscriber.id {
						registry.Deregister(sub.svc)
					}
					subscribers = append(subscribers, sub)
				}
				h.subscribers[subscriber.topic] = subscribers
				h.Unlock()
			}
		}
	}()

	h.running = true
	return nil
}

func (h *httpBroker) stop() error {
	ch := make(chan error)
	h.exit <- ch
	return <-ch
}

func (h *httpBroker) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		err := errors.BadRequest("go.micro.broker", "Method not allowed")
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
		return
	}
	defer req.Body.Close()

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		errr := errors.InternalServerError("go.micro.broker", fmt.Sprintf("Error reading request body: %v", err))
		w.WriteHeader(500)
		w.Write([]byte(errr.Error()))
		return
	}

	var e *envelope
	if err = json.Unmarshal(b, &e); err != nil {
		errr := errors.InternalServerError("go.micro.broker", fmt.Sprintf("Error parsing request body: %v", err))
		w.WriteHeader(500)
		w.Write([]byte(errr.Error()))
		return
	}

	if len(e.Message.Topic) == 0 {
		errr := errors.InternalServerError("go.micro.broker", "Topic not found")
		w.WriteHeader(500)
		w.Write([]byte(errr.Error()))
		return
	}

	ctx := c.WithMetaData(context.Background(), e.Header)

	h.RLock()
	for _, subscriber := range h.subscribers[e.Message.Topic] {
		subscriber.fn(ctx, e.Message)
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

func (h *httpBroker) Init() error {
	if len(h.id) == 0 {
		h.id = "broker-" + uuid.NewUUID().String()
	}

	http.Handle(DefaultSubPath, h)
	return nil
}

func (h *httpBroker) Publish(ctx context.Context, topic string, body []byte) error {
	s, err := registry.GetService("topic:" + topic)
	if err != nil {
		return err
	}

	message := &Message{
		Id:        uuid.NewUUID().String(),
		Timestamp: time.Now().Unix(),
		Topic:     topic,
		Body:      body,
	}

	header, _ := c.GetMetaData(ctx)

	b, err := json.Marshal(&envelope{
		header,
		message,
	})

	if err != nil {
		return err
	}

	for _, node := range s.Nodes() {
		r, err := http.Post(fmt.Sprintf("http://%s:%d%s", node.Address(), node.Port(), DefaultSubPath), "application/json", bytes.NewBuffer(b))
		if err == nil {
			r.Body.Close()
		}
	}

	return nil
}

func (h *httpBroker) Subscribe(topic string, function func(context.Context, *Message)) (Subscriber, error) {
	// parse address for host, port
	parts := strings.Split(h.Address(), ":")
	host := strings.Join(parts[:len(parts)-1], ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	// register service
	node := registry.NewNode(h.id, host, port)
	service := registry.NewService("topic:"+topic, node)

	subscriber := &httpSubscriber{
		id:    uuid.NewUUID().String(),
		topic: topic,
		ch:    h.unsubscribe,
		fn:    function,
		svc:   service,
	}

	log.Infof("Registering subscriber %s", node.Id())
	if err := registry.Register(service); err != nil {
		return nil, err
	}

	h.Lock()
	h.subscribers[topic] = append(h.subscribers[topic], subscriber)
	h.Unlock()

	return subscriber, nil
}
