package nats

import (
	"encoding/json"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/apcera/nats"
	"github.com/myodc/go-micro/broker"
	c "github.com/myodc/go-micro/context"

	"golang.org/x/net/context"
)

type nbroker struct {
	addrs []string
	conn  *nats.Conn
}

type subscriber struct {
	s *nats.Subscription
}

// used in brokers where there is no support for headers
type envelope struct {
	Header  map[string]string
	Message *broker.Message
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

func (n *nbroker) Publish(ctx context.Context, topic string, body []byte) error {
	header, _ := c.GetMetadata(ctx)

	message := &broker.Message{
		Id:        uuid.NewUUID().String(),
		Timestamp: time.Now().Unix(),
		Topic:     topic,
		Body:      body,
	}

	b, err := json.Marshal(&envelope{
		header,
		message,
	})
	if err != nil {
		return err
	}
	return n.conn.Publish(topic, b)
}

func (n *nbroker) Subscribe(topic string, function func(context.Context, *broker.Message)) (broker.Subscriber, error) {
	sub, err := n.conn.Subscribe(topic, func(msg *nats.Msg) {
		var e *envelope
		if err := json.Unmarshal(msg.Data, &e); err != nil {
			return
		}
		ctx := c.WithMetadata(context.Background(), e.Header)
		function(ctx, e.Message)
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
