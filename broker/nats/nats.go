package nats

import (
	"encoding/json"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/apcera/nats"
	"github.com/myodc/go-micro/broker"
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

func (n *nbroker) Publish(topic string, data []byte) error {
	b, err := json.Marshal(&broker.Message{
		Id:        uuid.NewUUID().String(),
		Timestamp: time.Now().Unix(),
		Topic:     topic,
		Data:      data,
	})
	if err != nil {
		return err
	}
	return n.conn.Publish(topic, b)
}

func (n *nbroker) Subscribe(topic string, function func(*broker.Message)) (broker.Subscriber, error) {
	sub, err := n.conn.Subscribe(topic, func(msg *nats.Msg) {
		var data *broker.Message
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		function(data)
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
