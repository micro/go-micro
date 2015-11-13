package nats

import (
	"encoding/json"
	"strings"

	"github.com/apcera/nats"
	"github.com/myodc/go-micro/broker"
)

type nbroker struct {
	addrs []string
	conn  *nats.Conn
}

type subscriber struct {
	topic   string
	s       *nats.Subscription
	conn    *nats.Conn
	handler *natsMsgHandler
}

type natsMsgHandler struct {
	handlerFunc broker.HandlerFunc
}

func (n *natsMsgHandler) Handle(msg *broker.Message) error {
	return n.handlerFunc(msg)
}

func (n *natsMsgHandler) Ack(msg *broker.Message) error {
	// No need to acknowledge the message in Nats?
	return nil
}

func (n *subscriber) Topic() string {
	return n.s.Subject
}

func (n *subscriber) Name() string {
	return ""
}

func (n *subscriber) Unsubscribe() error {
	return n.s.Unsubscribe()
}

func (n *subscriber) SetHandlerFunc(h broker.HandlerFunc, concurrency int) {
	n.handler = &natsMsgHandler{
		handlerFunc: h,
	}
}

func (n *subscriber) Subscribe() error {
	sub, err := n.conn.Subscribe(n.topic, func(msg *nats.Msg) {
		m := new(broker.Message)
		if err := json.Unmarshal(msg.Data, m); err != nil {
			return
		}
		n.handler.Handle(m)
	})

	if err != nil {
		return err
	}

	n.s = sub

	return nil
}

func (n *nbroker) Address() string {
	if len(n.addrs) > 0 {
		return n.addrs[0]
	}
	return ""
}

func (n *nbroker) Connect() error {
	if n.conn != nil {
		return nil
	}

	opts := nats.DefaultOptions
	opts.Servers = n.addrs
	c, err := opts.Connect()
	if err != nil {
		return err
	}
	n.conn = c
	return nil
}

func (n *nbroker) Disconnect() error {
	n.conn.Close()
	return nil
}

func (n *nbroker) Init() error {
	return nil
}

func (n *nbroker) Publish(topic string, msg *broker.Message) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return n.conn.Publish(topic, b)
}

func (n *nbroker) NewSubscriber(name, topic string) (broker.Subscriber, error) {
	return &subscriber{
		topic: topic,
		conn:  n.conn,
	}, nil
}

// NewBroker instantiates and returns a newly created Nats-backed broker
func NewBroker(addrs []string, opt ...broker.Option) broker.Broker {
	var cAddrs []string
	for _, addr := range addrs {
		if len(addr) == 0 {
			continue
		}
		if !strings.HasPrefix(addr, "nats://") {
			addr = "nats://" + addr
		}
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{nats.DefaultURL}
	}
	return &nbroker{
		addrs: cAddrs,
	}
}
