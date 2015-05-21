package transport

import (
	"fmt"
	"sync"
	"time"

	"errors"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/streadway/amqp"
)

type RabbitMQTransport struct {
	conn  *rabbitMQConn
	addrs []string
}

type RabbitMQTransportClient struct {
	once    sync.Once
	rt      *RabbitMQTransport
	addr    string
	replyTo string

	sync.Mutex
	inflight map[string]chan amqp.Delivery
}

type RabbitMQTransportSocket struct {
	conn *rabbitMQConn
	d    *amqp.Delivery
}

type RabbitMQTransportListener struct {
	conn *rabbitMQConn
	addr string
}

func (r *RabbitMQTransportClient) init() {
	<-r.rt.conn.Init()
	if err := r.rt.conn.Channel.DeclareReplyQueue(r.replyTo); err != nil {
		return
	}
	deliveries, err := r.rt.conn.Channel.ConsumeQueue(r.replyTo)
	if err != nil {
		return
	}
	go func() {
		for delivery := range deliveries {
			go r.handle(delivery)
		}
	}()
}

func (r *RabbitMQTransportClient) handle(delivery amqp.Delivery) {
	ch := r.getReq(delivery.CorrelationId)
	if ch == nil {
		return
	}
	select {
	case ch <- delivery:
	default:
	}
}

func (r *RabbitMQTransportClient) putReq(id string) chan amqp.Delivery {
	r.Lock()
	ch := make(chan amqp.Delivery, 1)
	r.inflight[id] = ch
	r.Unlock()
	return ch
}

func (r *RabbitMQTransportClient) getReq(id string) chan amqp.Delivery {
	r.Lock()
	defer r.Unlock()
	if ch, ok := r.inflight[id]; ok {
		delete(r.inflight, id)
		return ch
	}
	return nil
}

func (r *RabbitMQTransportClient) Send(m *Message) (*Message, error) {
	r.once.Do(r.init)

	if !r.rt.conn.IsConnected() {
		return nil, errors.New("Not connected to AMQP")
	}

	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	replyChan := r.putReq(id.String())

	headers := amqp.Table{}

	for k, v := range m.Header {
		headers[k] = v
	}

	message := amqp.Publishing{
		CorrelationId: id.String(),
		Timestamp:     time.Now().UTC(),
		Body:          m.Body,
		ReplyTo:       r.replyTo,
		Headers:       headers,
	}

	if err := r.rt.conn.Publish("micro", r.addr, message); err != nil {
		r.getReq(id.String())
		return nil, err
	}

	select {
	case d := <-replyChan:
		mr := &Message{
			Header: make(map[string]string),
			Body:   d.Body,
		}

		for k, v := range d.Headers {
			mr.Header[k] = fmt.Sprintf("%v", v)
		}

		return mr, nil
	case <-time.After(time.Second * 10):
		return nil, errors.New("timed out")
	}
}

func (r *RabbitMQTransportClient) Close() error {
	return nil
}

func (r *RabbitMQTransportSocket) Recv(m *Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	mr := &Message{
		Header: make(map[string]string),
		Body:   r.d.Body,
	}

	for k, v := range r.d.Headers {
		mr.Header[k] = fmt.Sprintf("%v", v)
	}

	*m = *mr
	return nil
}

func (r *RabbitMQTransportSocket) Send(m *Message) error {
	msg := amqp.Publishing{
		CorrelationId: r.d.CorrelationId,
		Timestamp:     time.Now().UTC(),
		Body:          m.Body,
		Headers:       amqp.Table{},
	}

	for k, v := range m.Header {
		msg.Headers[k] = v
	}

	return r.conn.Publish("", r.d.ReplyTo, msg)
}

func (r *RabbitMQTransportSocket) Close() error {
	return nil
}

func (r *RabbitMQTransportListener) Addr() string {
	return r.addr
}

func (r *RabbitMQTransportListener) Close() error {
	r.conn.Close()
	return nil
}

func (r *RabbitMQTransportListener) Accept(fn func(Socket)) error {
	deliveries, err := r.conn.Consume(r.addr)
	if err != nil {
		return err
	}

	handler := func(d amqp.Delivery) {
		fn(&RabbitMQTransportSocket{
			d:    &d,
			conn: r.conn,
		})
	}

	for d := range deliveries {
		go handler(d)
	}

	return nil
}

func (r *RabbitMQTransport) Dial(addr string) (Client, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	return &RabbitMQTransportClient{
		rt:       r,
		addr:     addr,
		inflight: make(map[string]chan amqp.Delivery),
		replyTo:  fmt.Sprintf("replyTo-%s", id.String()),
	}, nil
}

func (r *RabbitMQTransport) Listen(addr string) (Listener, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	conn := newRabbitMQConn("", r.addrs)
	<-conn.Init()

	return &RabbitMQTransportListener{
		addr: id.String(),
		conn: conn,
	}, nil
}

func NewRabbitMQTransport(addrs []string) *RabbitMQTransport {
	return &RabbitMQTransport{
		conn:  newRabbitMQConn("", addrs),
		addrs: addrs,
	}
}
