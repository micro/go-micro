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
	"github.com/myodc/go-micro/errors"
	"github.com/myodc/go-micro/registry"
)

type HttpBroker struct {
	id          string
	address     string
	unsubscribe chan *HttpSubscriber

	sync.RWMutex
	subscribers map[string][]*HttpSubscriber
	running     bool
	exit        chan chan error
}

type HttpSubscriber struct {
	id    string
	topic string
	ch    chan *HttpSubscriber
	fn    func(*Message)
	svc   registry.Service
}

var (
	SubPath = "/_sub"
)

func (h *HttpSubscriber) Topic() string {
	return h.topic
}

func (h *HttpSubscriber) Unsubscribe() error {
	h.ch <- h
	return nil
}

func (h *HttpBroker) start() error {
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
				var subscribers []*HttpSubscriber
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

func (h *HttpBroker) stop() error {
	ch := make(chan error)
	h.exit <- ch
	return <-ch
}

func (h *HttpBroker) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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

	var msg *Message
	if err = json.Unmarshal(b, &msg); err != nil {
		errr := errors.InternalServerError("go.micro.broker", fmt.Sprintf("Error parsing request body: %v", err))
		w.WriteHeader(500)
		w.Write([]byte(errr.Error()))
		return
	}

	if len(msg.Topic) == 0 {
		errr := errors.InternalServerError("go.micro.broker", "Topic not found")
		w.WriteHeader(500)
		w.Write([]byte(errr.Error()))
		return
	}

	h.RLock()
	for _, subscriber := range h.subscribers[msg.Topic] {
		subscriber.fn(msg)
	}
	h.RUnlock()
}

func (h *HttpBroker) Address() string {
	return h.address
}

func (h *HttpBroker) Connect() error {
	return h.start()
}

func (h *HttpBroker) Disconnect() error {
	return h.stop()
}

func (h *HttpBroker) Init() error {
	if len(h.id) == 0 {
		h.id = "broker-" + uuid.NewUUID().String()
	}

	http.Handle(SubPath, h)
	return nil
}

func (h *HttpBroker) Publish(topic string, data []byte) error {
	s, err := registry.GetService("topic:" + topic)
	if err != nil {
		return err
	}

	b, err := json.Marshal(&Message{
		Id:        uuid.NewUUID().String(),
		Timestamp: time.Now().Unix(),
		Topic:     topic,
		Data:      data,
	})
	if err != nil {
		return err
	}

	for _, node := range s.Nodes() {
		r, err := http.Post(fmt.Sprintf("http://%s:%d%s", node.Address(), node.Port(), SubPath), "application/json", bytes.NewBuffer(b))
		if err == nil {
			r.Body.Close()
		}
	}

	return nil
}

func (h *HttpBroker) Subscribe(topic string, function func(*Message)) (Subscriber, error) {
	// parse address for host, port
	parts := strings.Split(h.Address(), ":")
	host := strings.Join(parts[:len(parts)-1], ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	// register service
	node := registry.NewNode(h.id, host, port)
	service := registry.NewService("topic:"+topic, node)

	subscriber := &HttpSubscriber{
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

func NewHttpBroker(address string) Broker {
	return &HttpBroker{
		id:          Id,
		address:     address,
		subscribers: make(map[string][]*HttpSubscriber),
		unsubscribe: make(chan *HttpSubscriber),
		exit:        make(chan chan error),
	}
}
