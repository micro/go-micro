package rabbitmq

import (
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/myodc/go-micro/broker"
	c "github.com/myodc/go-micro/context"
	"github.com/streadway/amqp"
	"golang.org/x/net/context"
)

type rbroker struct {
	conn  *rabbitMQConn
	addrs []string
}

type subscriber struct {
	topic string
	ch    *rabbitMQChannel
}

func (s *subscriber) Topic() string {
	return s.topic
}

func (s *subscriber) Unsubscribe() error {
	return s.ch.Close()
}

func (r *rbroker) Publish(ctx context.Context, topic string, body []byte) error {
	header, _ := c.GetMetadata(ctx)

	msg := amqp.Publishing{
		MessageId: uuid.NewUUID().String(),
		Timestamp: time.Now().UTC(),
		Body:      body,
		Headers:   amqp.Table{},
	}

	for k, v := range header {
		msg.Headers[k] = v
	}

	return r.conn.Publish("", topic, msg)
}

func (r *rbroker) Subscribe(topic string, function func(context.Context, *broker.Message)) (broker.Subscriber, error) {
	ch, sub, err := r.conn.Consume(topic)
	if err != nil {
		return nil, err
	}

	fn := func(msg amqp.Delivery) {
		header := make(map[string]string)
		for k, v := range msg.Headers {
			header[k], _ = v.(string)
		}
		ctx := c.WithMetadata(context.Background(), header)
		function(ctx, &broker.Message{
			Id:        msg.MessageId,
			Timestamp: msg.Timestamp.Unix(),
			Topic:     topic,
			Body:      msg.Body,
		})
	}

	go func() {
		for d := range sub {
			go fn(d)
		}
	}()

	return &subscriber{ch: ch, topic: topic}, nil
}

func (r *rbroker) Address() string {
	if len(r.addrs) > 0 {
		return r.addrs[0]
	}
	return ""
}

func (r *rbroker) Init() error {
	return nil
}

func (r *rbroker) Connect() error {
	<-r.conn.Init()
	return nil
}

func (r *rbroker) Disconnect() error {
	r.conn.Close()
	return nil
}

func NewBroker(addrs []string, opt ...broker.Option) broker.Broker {
	return &rbroker{
		conn:  newRabbitMQConn("", addrs),
		addrs: addrs,
	}
}
