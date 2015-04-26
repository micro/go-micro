package broker

import (
	"encoding/json"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/apcera/nats"
)

type NatsBroker struct {
	address string
	conn    *nats.Conn
}

type NatsSubscriber struct {
	s *nats.Subscription
}

func (n *NatsSubscriber) Topic() string {
	return n.s.Subject
}

func (n *NatsSubscriber) Unsubscribe() error {
	return n.s.Unsubscribe()
}

func (n *NatsBroker) Address() string {
	return n.address
}

func (n *NatsBroker) Connect() error {
	if n.conn != nil {
		return nil
	}

	c, err := nats.Connect(n.address)
	if err != nil {
		return err
	}
	n.conn = c
	return nil
}

func (n *NatsBroker) Disconnect() error {
	n.conn.Close()
	return nil
}

func (n *NatsBroker) Init() error {
	return nil
}

func (n *NatsBroker) Publish(topic string, data []byte) error {
	b, err := json.Marshal(&Message{
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

func (n *NatsBroker) Subscribe(topic string, function func(*Message)) (Subscriber, error) {
	subscriber, err := n.conn.Subscribe(topic, func(msg *nats.Msg) {
		var data *Message
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		function(data)
	})
	if err != nil {
		return nil, err
	}
	return &NatsSubscriber{s: subscriber}, nil
}

func NewNatsBroker(address string) Broker {
	return &NatsBroker{
		address: address,
	}
}
