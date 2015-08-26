package rabbitmq

import (
	"fmt"
	"sync"
	"time"

	"errors"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/streadway/amqp"

	"github.com/kynrai/go-micro/transport"
)

type rmqtport struct {
	conn  *rabbitMQConn
	addrs []string

	once    sync.Once
	replyTo string

	sync.Mutex
	inflight map[string]chan amqp.Delivery
}

type rmqtportClient struct {
	rt    *rmqtport
	addr  string
	corId string
	reply chan amqp.Delivery
}

type rmqtportSocket struct {
	conn *rabbitMQConn
	d    *amqp.Delivery
}

type rmqtportListener struct {
	conn *rabbitMQConn
	addr string
}

func (r *rmqtportClient) Send(m *transport.Message) error {
	if !r.rt.conn.IsConnected() {
		return errors.New("Not connected to AMQP")
	}

	headers := amqp.Table{}
	for k, v := range m.Header {
		headers[k] = v
	}

	message := amqp.Publishing{
		CorrelationId: r.corId,
		Timestamp:     time.Now().UTC(),
		Body:          m.Body,
		ReplyTo:       r.rt.replyTo,
		Headers:       headers,
	}

	if err := r.rt.conn.Publish("micro", r.addr, message); err != nil {
		return err
	}

	return nil
}

func (r *rmqtportClient) Recv(m *transport.Message) error {
	select {
	case d := <-r.reply:
		mr := &transport.Message{
			Header: make(map[string]string),
			Body:   d.Body,
		}

		for k, v := range d.Headers {
			mr.Header[k] = fmt.Sprintf("%v", v)
		}

		*m = *mr
		return nil
	case <-time.After(time.Second * 10):
		return errors.New("timed out")
	}
}

func (r *rmqtportClient) Close() error {
	r.rt.popReq(r.corId)
	return nil
}

func (r *rmqtportSocket) Recv(m *transport.Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	mr := &transport.Message{
		Header: make(map[string]string),
		Body:   r.d.Body,
	}

	for k, v := range r.d.Headers {
		mr.Header[k] = fmt.Sprintf("%v", v)
	}

	*m = *mr
	return nil
}

func (r *rmqtportSocket) Send(m *transport.Message) error {
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

func (r *rmqtportSocket) Close() error {
	return nil
}

func (r *rmqtportListener) Addr() string {
	return r.addr
}

func (r *rmqtportListener) Close() error {
	r.conn.Close()
	return nil
}

func (r *rmqtportListener) Accept(fn func(transport.Socket)) error {
	deliveries, err := r.conn.Consume(r.addr)
	if err != nil {
		return err
	}

	handler := func(d amqp.Delivery) {
		fn(&rmqtportSocket{
			d:    &d,
			conn: r.conn,
		})
	}

	for d := range deliveries {
		go handler(d)
	}

	return nil
}

func (r *rmqtport) putReq(id string) chan amqp.Delivery {
	r.Lock()
	ch := make(chan amqp.Delivery, 1)
	r.inflight[id] = ch
	r.Unlock()
	return ch
}

func (r *rmqtport) getReq(id string) chan amqp.Delivery {
	r.Lock()
	defer r.Unlock()
	if ch, ok := r.inflight[id]; ok {
		return ch
	}
	return nil
}

func (r *rmqtport) popReq(id string) {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.inflight[id]; ok {
		delete(r.inflight, id)
	}
}

func (r *rmqtport) init() {
	<-r.conn.Init()
	if err := r.conn.Channel.DeclareReplyQueue(r.replyTo); err != nil {
		return
	}
	deliveries, err := r.conn.Channel.ConsumeQueue(r.replyTo)
	if err != nil {
		return
	}
	go func() {
		for delivery := range deliveries {
			go r.handle(delivery)
		}
	}()
}

func (r *rmqtport) handle(delivery amqp.Delivery) {
	ch := r.getReq(delivery.CorrelationId)
	if ch == nil {
		return
	}
	ch <- delivery
}

func (r *rmqtport) Dial(addr string, opts ...transport.DialOption) (transport.Client, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	r.once.Do(r.init)

	return &rmqtportClient{
		rt:    r,
		addr:  addr,
		corId: id.String(),
		reply: r.putReq(id.String()),
	}, nil
}

func (r *rmqtport) Listen(addr string) (transport.Listener, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	conn := newRabbitMQConn("", r.addrs)
	<-conn.Init()

	return &rmqtportListener{
		addr: id.String(),
		conn: conn,
	}, nil
}

func NewTransport(addrs []string, opt ...transport.Option) transport.Transport {
	id, _ := uuid.NewV4()

	return &rmqtport{
		conn:     newRabbitMQConn("", addrs),
		addrs:    addrs,
		replyTo:  id.String(),
		inflight: make(map[string]chan amqp.Delivery),
	}
}
