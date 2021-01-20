// Package rabbitmq provides a RabbitMQ transport
package rabbitmq

import (
	"fmt"
	"io"
	"sync"
	"time"

	"errors"
	"github.com/google/uuid"
	"github.com/streadway/amqp"

	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/transport"
)

const (
	directReplyQueue = "amq.rabbitmq.reply-to"
)

type rmqtport struct {
	conn  *rabbitMQConn
	addrs []string
	opts  transport.Options

	once    sync.Once
	replyTo string

	sync.Mutex
	inflight map[string]chan amqp.Delivery
}

type rmqtportClient struct {
	rt     *rmqtport
	addr   string
	corId  string
	local  string
	remote string
	reply  chan amqp.Delivery
}

type rmqtportSocket struct {
	rt     *rmqtport
	conn   *rabbitMQConn
	d      *amqp.Delivery
	close  chan bool
	local  string
	remote string

	sync.Mutex
	r  chan *amqp.Delivery
	bl []*amqp.Delivery
}

type rmqtportListener struct {
	rt   *rmqtport
	conn *rabbitMQConn
	exit chan bool
	addr string

	sync.RWMutex
	so map[string]*rmqtportSocket
}

var (
	DefaultTimeout = time.Minute
)

func init() {
	cmd.DefaultTransports["rabbitmq"] = NewTransport
}

func (r *rmqtportClient) Local() string {
	return r.local
}

func (r *rmqtportClient) Remote() string {
	return r.remote
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

	// no timeout
	if r.rt.opts.Timeout == time.Duration(0) {
		return r.rt.conn.Publish(DefaultExchange, r.addr, message)
	}

	// use the timeout
	ch := make(chan error, 1)

	go func() {
		ch <- r.rt.conn.Publish(DefaultExchange, r.addr, message)
	}()

	select {
	case err := <-ch:
		return err
	case <-time.After(r.rt.opts.Timeout):
		return errors.New("timed out")
	}
}

func (r *rmqtportClient) Recv(m *transport.Message) error {
	timeout := DefaultTimeout
	if r.rt.opts.Timeout > time.Duration(0) {
		timeout = r.rt.opts.Timeout
	}

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
	case <-time.After(timeout):
		return errors.New("timed out")
	}
}

func (r *rmqtportClient) Close() error {
	r.rt.popReq(r.corId)
	return nil
}

func (r *rmqtportSocket) Local() string {
	return r.local
}

func (r *rmqtportSocket) Remote() string {
	return r.remote
}

func (r *rmqtportSocket) Recv(m *transport.Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	var d *amqp.Delivery
	var ok bool

	if r.rt.opts.Timeout > time.Duration(0) {
		select {
		case d, ok = <-r.r:
		case <-time.After(r.rt.opts.Timeout):
			return errors.New("timed out")
		}
	} else {
		d, ok = <-r.r
	}

	if !ok {
		return io.EOF
	}

	r.Lock()
	if len(r.bl) > 0 {
		select {
		case r.r <- r.bl[0]:
			r.bl = r.bl[1:]
		default:
		}
	}
	r.Unlock()

	mr := &transport.Message{
		Header: make(map[string]string),
		Body:   d.Body,
	}

	for k, v := range d.Headers {
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

	// no timeout
	if r.rt.opts.Timeout == time.Duration(0) {
		return r.conn.Publish("", r.d.ReplyTo, msg)
	}

	// use the timeout
	ch := make(chan error, 1)

	go func() {
		ch <- r.conn.Publish("", r.d.ReplyTo, msg)
	}()

	select {
	case err := <-ch:
		return err
	case <-time.After(r.rt.opts.Timeout):
		return errors.New("timed out")
	}
}

func (r *rmqtportSocket) Close() error {
	select {
	case <-r.close:
		return nil
	default:
		close(r.close)
	}
	return nil
}

func (r *rmqtportListener) Addr() string {
	return r.addr
}

func (r *rmqtportListener) Close() error {
	r.exit <- true
	r.conn.Close()
	return nil
}

func (r *rmqtportListener) Accept(fn func(transport.Socket)) error {
	for {
		// connect if not connected
		if !r.conn.IsConnected() {
			// reinitialise
			<-r.conn.Init(r.rt.opts.Secure, r.rt.opts.TLSConfig)
		}

		// accept connections
		exit, err := r.accept(fn)
		if err != nil {
			return err
		}

		// connection closed
		if exit {
			return nil
		}
	}
}

func (r *rmqtportListener) accept(fn func(transport.Socket)) (bool, error) {
	deliveries, err := r.conn.Consume(r.addr)
	if err != nil {
		return false, err
	}

	for {
		select {
		case <-r.exit:
			return true, nil
		case d, ok := <-deliveries:
			if !ok {
				return false, nil
			}

			r.RLock()
			sock, ok := r.so[d.CorrelationId]
			r.RUnlock()
			if !ok {
				sock = &rmqtportSocket{
					rt:     r.rt,
					d:      &d,
					r:      make(chan *amqp.Delivery, 1),
					conn:   r.conn,
					close:  make(chan bool, 1),
					local:  r.Addr(),
					remote: d.CorrelationId,
				}
				r.Lock()
				r.so[sock.d.CorrelationId] = sock
				r.Unlock()

				go func() {
					<-sock.close
					r.Lock()
					delete(r.so, sock.d.CorrelationId)
					r.Unlock()
				}()

				go fn(sock)
			}

			select {
			case <-sock.close:
				continue
			default:
			}

			sock.Lock()
			sock.bl = append(sock.bl, &d)
			select {
			case sock.r <- sock.bl[0]:
				sock.bl = sock.bl[1:]
			default:
			}
			sock.Unlock()
		}
	}

	return false, nil
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
	<-r.conn.Init(r.opts.Secure, r.opts.TLSConfig)
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
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	r.once.Do(r.init)

	return &rmqtportClient{
		rt:     r,
		addr:   addr,
		corId:  id.String(),
		reply:  r.putReq(id.String()),
		local:  id.String(),
		remote: addr,
	}, nil
}

func (r *rmqtport) Listen(addr string, opts ...transport.ListenOption) (transport.Listener, error) {
	if len(addr) == 0 || addr == ":0" {
		id, err := uuid.NewRandom()
		if err != nil {
			return nil, err
		}
		addr = id.String()
	}

	conn := newRabbitMQConn("", r.addrs)
	<-conn.Init(r.opts.Secure, r.opts.TLSConfig)

	return &rmqtportListener{
		rt:   r,
		addr: addr,
		conn: conn,
		exit: make(chan bool, 1),
		so:   make(map[string]*rmqtportSocket),
	}, nil
}

func (r *rmqtport) Init(opts ...transport.Option) error {
	for _, o := range opts {
		o(&r.opts)
	}
	r.addrs = r.opts.Addrs
	r.conn.Close()
	r.conn = newRabbitMQConn("", r.opts.Addrs)
	return nil
}

func (r *rmqtport) Options() transport.Options {
	return r.opts
}

func (r *rmqtport) String() string {
	return "rabbitmq"
}

func NewTransport(opts ...transport.Option) transport.Transport {
	options := transport.Options{
		Timeout: DefaultTimeout,
	}

	for _, o := range opts {
		o(&options)
	}

	return &rmqtport{
		opts:     options,
		conn:     newRabbitMQConn("", options.Addrs),
		addrs:    options.Addrs,
		replyTo:  directReplyQueue,
		inflight: make(map[string]chan amqp.Delivery),
	}
}
