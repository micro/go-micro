// Package rabbitmq provides a RabbitMQ broker
package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/streadway/amqp"
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/logger"
)

type rbroker struct {
	conn           *rabbitMQConn
	addrs          []string
	opts           broker.Options
	prefetchCount  int
	prefetchGlobal bool
	mtx            sync.Mutex
	wg             sync.WaitGroup
}

type subscriber struct {
	mtx          sync.Mutex
	unsub        chan bool
	opts         broker.SubscribeOptions
	topic        string
	ch           *rabbitMQChannel
	durableQueue bool
	queueArgs    map[string]interface{}
	r            *rbroker
	fn           func(msg amqp.Delivery)
	headers      map[string]interface{}
	wg           sync.WaitGroup
}

type publication struct {
	d   amqp.Delivery
	m   *broker.Message
	t   string
	err error
}



func (p *publication) Ack() error {
	return p.d.Ack(false)
}

func (p *publication) Error() error {
	return p.err
}

func (p *publication) Topic() string {
	return p.t
}

func (p *publication) Message() *broker.Message {
	return p.m
}

func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.topic
}

func (s *subscriber) Unsubscribe() error {
	s.unsub <- true

	// Need to wait on subscriber to exit if autoack is disabled
	// since closing the channel will prevent the ack/nack from
	// being sent upon handler completion.
	if !s.opts.AutoAck {
		s.wg.Wait()
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.ch != nil {
		return s.ch.Close()
	}
	return nil
}

func (s *subscriber) resubscribe() {
	s.wg.Add(1)
	defer s.wg.Done()

	minResubscribeDelay := 100 * time.Millisecond
	maxResubscribeDelay := 30 * time.Second
	expFactor := time.Duration(2)
	reSubscribeDelay := minResubscribeDelay
	// loop until unsubscribe
	for {
		select {
		// unsubscribe case
		case <-s.unsub:
			return
		// check shutdown case
		case <-s.r.conn.close:
			// yep, its shutdown case
			return
			// wait until we reconect to rabbit
		case <-s.r.conn.waitConnection:
			// When the connection is disconnected, the waitConnection will be re-assigned, so '<-s.r.conn.waitConnection' maybe blocked.
			// Here, it returns once a second, and then the latest waitconnection will be used
		case <-time.After(time.Second):
			continue
		}

		// it may crash (panic) in case of Consume without connection, so recheck it
		s.r.mtx.Lock()
		if !s.r.conn.connected {
			s.r.mtx.Unlock()
			continue
		}

		ch, sub, err := s.r.conn.Consume(
			s.opts.Queue,
			s.topic,
			s.headers,
			s.queueArgs,
			s.opts.AutoAck,
			s.durableQueue,
		)

		s.r.mtx.Unlock()
		switch err {
		case nil:
			reSubscribeDelay = minResubscribeDelay
			s.mtx.Lock()
			s.ch = ch
			s.mtx.Unlock()
		default:
			if reSubscribeDelay > maxResubscribeDelay {
				reSubscribeDelay = maxResubscribeDelay
			}
			time.Sleep(reSubscribeDelay)
			reSubscribeDelay *= expFactor
			continue
		}

	SubLoop:
		for {
			select {
			case <-s.unsub:
				return
			case d, ok := <-sub:
				if !ok {
					break SubLoop
				}
				s.r.wg.Add(1)
				s.fn(d)
				s.r.wg.Done()
			}
		}
	}
}

func (r *rbroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	m := amqp.Publishing{
		Body:    msg.Body,
		Headers: amqp.Table{},
	}

	options := broker.PublishOptions{}
	for _, o := range opts {
		o(&options)
	}

	if options.Context != nil {
		if value, ok := options.Context.Value(deliveryMode{}).(uint8); ok {
			m.DeliveryMode = value
		}

		if value, ok := options.Context.Value(priorityKey{}).(uint8); ok {
			m.Priority = value
		}

		if value, ok := options.Context.Value(contentType{}).(string); ok {
			m.Headers["Content-Type"] = value
			m.ContentType = value
		}

		if value, ok := options.Context.Value(contentEncoding{}).(string); ok {
			m.ContentEncoding = value
		}

		if value, ok := options.Context.Value(correlationID{}).(string); ok {
			m.CorrelationId = value
		}

		if value, ok := options.Context.Value(replyTo{}).(string); ok {
			m.ReplyTo = value
		}

		if value, ok := options.Context.Value(expiration{}).(string); ok {
			m.Expiration = value
		}

		if value, ok := options.Context.Value(messageID{}).(string); ok {
			m.MessageId = value
		}

		if value, ok := options.Context.Value(timestamp{}).(time.Time); ok {
			m.Timestamp = value
		}

		if value, ok := options.Context.Value(typeMsg{}).(string); ok {
			m.Type = value
		}

		if value, ok := options.Context.Value(userID{}).(string); ok {
			m.UserId = value
		}

		if value, ok := options.Context.Value(appID{}).(string); ok {
			m.AppId = value
		}
	}

	for k, v := range msg.Header {
		m.Headers[k] = v
	}

	if r.getWithoutExchange() {
		m.Headers["Micro-Topic"] = topic
	}

	if r.conn == nil {
		return errors.New("connection is nil")
	}

	return r.conn.Publish(r.conn.exchange.Name, topic, m)
}

func (r *rbroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	var ackSuccess bool

	if r.conn == nil {
		return nil, errors.New("not connected")
	}

	opt := broker.SubscribeOptions{
		AutoAck: true,
	}

	for _, o := range opts {
		o(&opt)
	}

	// Make sure context is setup
	if opt.Context == nil {
		opt.Context = context.Background()
	}

	ctx := opt.Context
	if subscribeContext, ok := ctx.Value(subscribeContextKey{}).(context.Context); ok && subscribeContext != nil {
		ctx = subscribeContext
	}

	var requeueOnError bool
	requeueOnError, _ = ctx.Value(requeueOnErrorKey{}).(bool)

	var durableQueue bool
	durableQueue, _ = ctx.Value(durableQueueKey{}).(bool)

	var qArgs map[string]interface{}
	if qa, ok := ctx.Value(queueArgumentsKey{}).(map[string]interface{}); ok {
		qArgs = qa
	}

	var headers map[string]interface{}
	if h, ok := ctx.Value(headersKey{}).(map[string]interface{}); ok {
		headers = h
	}

	if bval, ok := ctx.Value(ackSuccessKey{}).(bool); ok && bval {
		opt.AutoAck = false
		ackSuccess = true
	}

	fn := func(msg amqp.Delivery) {
		header := make(map[string]string)
		for k, v := range msg.Headers {
			header[k] = fmt.Sprintf("%v", v)
		}

		// Get rid of dependence on 'Micro-Topic'
		msgTopic := header["Micro-Topic"]
		if msgTopic == "" {
			header["Micro-Topic"] = msg.RoutingKey
		}

		m := &broker.Message{
			Header: header,
			Body:   msg.Body,
		}
		p := &publication{d: msg, m: m, t: msg.RoutingKey}
		p.err = handler(p)
		if p.err == nil && ackSuccess && !opt.AutoAck {
			msg.Ack(false)
		} else if p.err != nil && !opt.AutoAck {
			msg.Nack(false, requeueOnError)
		}
	}

	sret := &subscriber{topic: topic, opts: opt, unsub: make(chan bool), r: r,
		durableQueue: durableQueue, fn: fn, headers: headers, queueArgs: qArgs,
		wg: sync.WaitGroup{}}

	go sret.resubscribe()

	return sret, nil
}

func (r *rbroker) Options() broker.Options {
	return r.opts
}

func (r *rbroker) String() string {
	return "rabbitmq"
}

func (r *rbroker) Address() string {
	if len(r.addrs) > 0 {
		u, err := url.Parse(r.addrs[0])
		if err != nil {
			return ""
		}

		return u.Redacted()
	}
	return ""
}

func (r *rbroker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&r.opts)
	}
	r.addrs = r.opts.Addrs
	return nil
}

func (r *rbroker) Connect() error {
	if r.conn == nil {
		r.conn = newRabbitMQConn(
			r.getExchange(),
			r.opts.Addrs,
			r.getPrefetchCount(),
			r.getPrefetchGlobal(),
			r.getConfirmPublish(),
			r.getWithoutExchange(),
			r.opts.Logger,
		)
	}

	conf := defaultAmqpConfig

	if auth, ok := r.opts.Context.Value(externalAuth{}).(ExternalAuthentication); ok {
		conf.SASL = []amqp.Authentication{&auth}
	}

	conf.TLSClientConfig = r.opts.TLSConfig

	return r.conn.Connect(r.opts.Secure, &conf)
}

func (r *rbroker) Disconnect() error {
	if r.conn == nil {
		return errors.New("connection is nil")
	}
	ret := r.conn.Close()
	r.wg.Wait() // wait all goroutines
	return ret
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.Options{
		Context: context.Background(),
		Logger:  logger.DefaultLogger,
	}

	for _, o := range opts {
		o(&options)
	}

	return &rbroker{
		addrs: options.Addrs,
		opts:  options,
	}
}

func (r *rbroker) getExchange() Exchange {
	ex := DefaultExchange

	if e, ok := r.opts.Context.Value(exchangeKey{}).(string); ok {
		ex.Name = e
	}

	if t, ok := r.opts.Context.Value(exchangeTypeKey{}).(MQExchangeType); ok {
		ex.Type = t
	}

	if d, ok := r.opts.Context.Value(durableExchange{}).(bool); ok {
		ex.Durable = d
	}

	return ex
}

func (r *rbroker) getPrefetchCount() int {
	if e, ok := r.opts.Context.Value(prefetchCountKey{}).(int); ok {
		return e
	}
	return DefaultPrefetchCount
}

func (r *rbroker) getPrefetchGlobal() bool {
	if e, ok := r.opts.Context.Value(prefetchGlobalKey{}).(bool); ok {
		return e
	}
	return DefaultPrefetchGlobal
}

func (r *rbroker) getConfirmPublish() bool {
	if e, ok := r.opts.Context.Value(confirmPublishKey{}).(bool); ok {
		return e
	}
	return DefaultConfirmPublish
}

func (r *rbroker) getWithoutExchange() bool {
	if e, ok := r.opts.Context.Value(withoutExchangeKey{}).(bool); ok {
		return e
	}
	return DefaultWithoutExchange
}
