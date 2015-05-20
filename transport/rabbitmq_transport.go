package transport

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"errors"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/streadway/amqp"
)

type RabbitMQTransport struct {
	conn *RabbitConnection
}

type RabbitMQTransportClient struct {
	once    sync.Once
	conn    *RabbitConnection
	target  string
	replyTo string

	sync.Mutex
	inflight map[string]chan amqp.Delivery
}

type RabbitMQTransportSocket struct {
	d   *amqp.Delivery
	hdr amqp.Table
	buf *bytes.Buffer
}

type RabbitMQTransportServer struct {
	conn *RabbitConnection
	name string
}

func (h *RabbitMQTransportClient) init() {
	<-h.conn.Init()
	if err := h.conn.Channel.DeclareReplyQueue(h.replyTo); err != nil {
		return
	}
	deliveries, err := h.conn.Channel.ConsumeQueue(h.replyTo)
	if err != nil {
		return
	}
	go func() {
		for delivery := range deliveries {
			go h.handle(delivery)
		}
	}()
}

func (h *RabbitMQTransportClient) handle(delivery amqp.Delivery) {
	ch := h.getReq(delivery.CorrelationId)
	if ch == nil {
		return
	}
	select {
	case ch <- delivery:
	default:
	}
}

func (h *RabbitMQTransportClient) putReq(id string) chan amqp.Delivery {
	h.Lock()
	ch := make(chan amqp.Delivery, 1)
	h.inflight[id] = ch
	h.Unlock()
	return ch
}

func (h *RabbitMQTransportClient) getReq(id string) chan amqp.Delivery {
	h.Lock()
	defer h.Unlock()
	if ch, ok := h.inflight[id]; ok {
		delete(h.inflight, id)
		return ch
	}
	return nil
}

func (h *RabbitMQTransportClient) Send(m *Message) (*Message, error) {
	h.once.Do(h.init)

	if !h.conn.IsConnected() {
		return nil, errors.New("Not connected to AMQP")
	}

	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	replyChan := h.putReq(id.String())

	headers := amqp.Table{}

	for k, v := range m.Header {
		headers[k] = v
	}

	message := amqp.Publishing{
		CorrelationId: id.String(),
		Timestamp:     time.Now().UTC(),
		Body:          m.Body,
		ReplyTo:       h.replyTo,
		Headers:       headers,
	}

	if err := h.conn.Publish("micro", h.target, message); err != nil {
		h.getReq(id.String())
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

func (h *RabbitMQTransportClient) Close() error {
	h.conn.Close()
	return nil
}

func (h *RabbitMQTransportSocket) Recv() (*Message, error) {
	m := &Message{
		Header: make(map[string]string),
		Body:   h.d.Body,
	}

	for k, v := range h.d.Headers {
		m.Header[k] = fmt.Sprintf("%v", v)
	}

	return m, nil
}

func (h *RabbitMQTransportSocket) WriteHeader(k string, v string) {
	h.hdr[k] = v
}

func (h *RabbitMQTransportSocket) Write(b []byte) error {
	_, err := h.buf.Write(b)
	return err
}

func (h *RabbitMQTransportServer) Addr() string {
	return h.conn.Connection.LocalAddr().String()
}

func (h *RabbitMQTransportServer) Close() error {
	h.conn.Close()
	return nil
}

func (h *RabbitMQTransportServer) Serve(fn func(Socket)) error {
	deliveries, err := h.conn.Consume(h.name)
	if err != nil {
		return err
	}

	handler := func(d amqp.Delivery) {
		buf := bytes.NewBuffer(nil)
		headers := amqp.Table{}

		fn(&RabbitMQTransportSocket{
			d:   &d,
			hdr: headers,
			buf: buf,
		})

		msg := amqp.Publishing{
			CorrelationId: d.CorrelationId,
			Timestamp:     time.Now().UTC(),
			Body:          buf.Bytes(),
			Headers:       headers,
		}

		h.conn.Publish("", d.ReplyTo, msg)
		buf.Reset()
	}

	for d := range deliveries {
		go handler(d)
	}

	return nil
}

func (h *RabbitMQTransport) NewClient(name, addr string) (Client, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	return &RabbitMQTransportClient{
		conn:     h.conn,
		target:   name,
		inflight: make(map[string]chan amqp.Delivery),
		replyTo:  fmt.Sprintf("replyTo-%s", id.String()),
	}, nil
}

func (h *RabbitMQTransport) NewServer(name, addr string) (Server, error) {
	conn := NewRabbitConnection("", "")
	<-conn.Init()

	return &RabbitMQTransportServer{
		name: name,
		conn: conn,
	}, nil
}

func NewRabbitMQTransport(addrs []string) *RabbitMQTransport {
	return &RabbitMQTransport{
		conn: NewRabbitConnection("", ""),
	}
}
