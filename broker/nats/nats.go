package nats

import (
	"encoding/json"
	"strings"

	"github.com/apcera/nats"
	"github.com/kynrai/go-micro/broker"
)

type nbroker struct {
	addrs []string
	conn  *nats.Conn
}

type subscriber struct {
	s *nats.Subscription
}

func (n *subscriber) Topic() string {
	return n.s.Subject
}

func (n *subscriber) Unsubscribe() error {
	return n.s.Unsubscribe()
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

func (n *nbroker) Subscribe(topic string, handler broker.Handler) (broker.Subscriber, error) {
	sub, err := n.conn.Subscribe(topic, func(msg *nats.Msg) {
		var m *broker.Message
		if err := json.Unmarshal(msg.Data, &m); err != nil {
			return
		}
		handler(m)
	})
	if err != nil {
		return nil, err
	}
	return &subscriber{s: sub}, nil
}

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
