package rabbitmq

import (
	"github.com/myodc/go-micro/broker"
	"github.com/streadway/amqp"
)

type rbroker struct {
	conn  *rabbitMQConn
	addrs []string
}

type subscriber struct {
	topic        string
	conn         *rabbitMQConn
	ch           *rabbitMQChannel
	handler      *amqpMsgHandler
	handlerCount int
	stopChan     chan bool
}

type amqpMsgHandler struct {
	handlerFunc broker.HandlerFunc
}

func (h *amqpMsgHandler) Handle(msg *broker.Message) error {
	return h.handlerFunc(msg)
}

func (h *amqpMsgHandler) Ack(msg *broker.Message) error {
	// Nothing to do regarding manual ack-ing of messages in AMQP world
	return nil
}

func (s *subscriber) Topic() string {
	return s.topic
}

func (s *subscriber) Name() string {
	return ""
}

func (s *subscriber) SetHandlerFunc(h broker.HandlerFunc, concurrency int) {
	s.handler = &amqpMsgHandler{
		handlerFunc: h,
	}
	s.handlerCount = concurrency
}

func (s *subscriber) Subscribe() error {

	ch, sub, err := s.conn.Consume(s.topic)
	if err != nil {
		return err
	}

	s.ch = ch

	for i := 0; i < s.handlerCount; i++ {
		go func() {
			for {
				select {
				case msg := <-sub:
					// Reconstitute the message header
					header := make(map[string]string)
					for k, v := range msg.Headers {
						header[k], _ = v.(string)
					}

					s.handler.Handle(&broker.Message{
						Header: header,
						Body:   msg.Body,
					})

				case <-s.stopChan:
					// We must have closed the subscription
					return
				}
			}
		}()
	}

	return nil
}

func (s *subscriber) Unsubscribe() error {
	if err := s.ch.Close(); err != nil {
		return err
	}

	close(s.stopChan)
	return nil
}

func (r *rbroker) Publish(topic string, msg *broker.Message) error {
	m := amqp.Publishing{
		Body:    msg.Body,
		Headers: amqp.Table{},
	}

	for k, v := range msg.Header {
		m.Headers[k] = v
	}

	return r.conn.Publish("", topic, m)
}

func (r *rbroker) NewSubscriber(name string, topic string) (broker.Subscriber, error) {
	return &subscriber{
		topic:    topic,
		conn:     r.conn,
		stopChan: make(chan bool),
	}, nil
}

func (r *rbroker) Subscribe(topic, name string, handlerFunc func(*broker.Message) error) (broker.Subscriber, error) {
	ch, sub, err := r.conn.Consume(topic)
	if err != nil {
		return nil, err
	}

	fn := func(msg amqp.Delivery) {
		header := make(map[string]string)
		for k, v := range msg.Headers {
			header[k], _ = v.(string)
		}
		handlerFunc(&broker.Message{
			Header: header,
			Body:   msg.Body,
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

// NewBroker instantiates and returns a newly created AMQP-backed broker
func NewBroker(addrs []string, opt ...broker.Option) broker.Broker {
	return &rbroker{
		conn:  newRabbitMQConn("", addrs),
		addrs: addrs,
	}
}
