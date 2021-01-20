// Package nsq provides an NSQ broker
package nsq

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/asim/go-micro/v3/broker"
	"github.com/asim/go-micro/v3/codec/json"
	"github.com/asim/go-micro/v3/cmd"
	"github.com/nsqio/go-nsq"
)

type nsqBroker struct {
	lookupdAddrs []string
	addrs        []string
	opts         broker.Options
	config       *nsq.Config

	sync.Mutex
	running bool
	p       []*nsq.Producer
	c       []*subscriber
}

type publication struct {
	topic string
	m     *broker.Message
	nm    *nsq.Message
	opts  broker.PublishOptions
	err   error
}

type subscriber struct {
	topic string
	opts  broker.SubscribeOptions

	c *nsq.Consumer

	// handler so we can resubcribe
	h nsq.HandlerFunc
	// concurrency
	n int
}

var (
	DefaultConcurrentHandlers = 1
)

func init() {
	rand.Seed(time.Now().UnixNano())
	cmd.DefaultBrokers["nsq"] = NewBroker
}

func (n *nsqBroker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&n.opts)
	}

	var addrs []string

	for _, addr := range n.opts.Addrs {
		if len(addr) > 0 {
			addrs = append(addrs, addr)
		}
	}

	if len(addrs) == 0 {
		addrs = []string{"127.0.0.1:4150"}
	}

	n.addrs = addrs
	n.configure(n.opts.Context)
	return nil
}

func (n *nsqBroker) configure(ctx context.Context) {
	if v, ok := ctx.Value(lookupdAddrsKey{}).([]string); ok {
		n.lookupdAddrs = v
	}

	if v, ok := ctx.Value(consumerOptsKey{}).([]string); ok {
		cfgFlag := &nsq.ConfigFlag{Config: n.config}
		for _, opt := range v {
			cfgFlag.Set(opt)
		}
	}
}

func (n *nsqBroker) Options() broker.Options {
	return n.opts
}

func (n *nsqBroker) Address() string {
	return n.addrs[rand.Intn(len(n.addrs))]
}

func (n *nsqBroker) Connect() error {
	n.Lock()
	defer n.Unlock()

	if n.running {
		return nil
	}

	producers := make([]*nsq.Producer, 0, len(n.addrs))

	// create producers
	for _, addr := range n.addrs {
		p, err := nsq.NewProducer(addr, n.config)
		if err != nil {
			return err
		}
		if err = p.Ping(); err != nil {
			return err
		}
		producers = append(producers, p)
	}

	// create consumers
	for _, c := range n.c {
		channel := c.opts.Queue
		if len(channel) == 0 {
			channel = uuid.New().String() + "#ephemeral"
		}

		cm, err := nsq.NewConsumer(c.topic, channel, n.config)
		if err != nil {
			return err
		}

		cm.AddConcurrentHandlers(c.h, c.n)

		c.c = cm

		if len(n.lookupdAddrs) > 0 {
			c.c.ConnectToNSQLookupds(n.lookupdAddrs)
		} else {
			err = c.c.ConnectToNSQDs(n.addrs)
			if err != nil {
				return err
			}
		}
	}

	n.p = producers
	n.running = true
	return nil
}

func (n *nsqBroker) Disconnect() error {
	n.Lock()
	defer n.Unlock()

	if !n.running {
		return nil
	}

	// stop the producers
	for _, p := range n.p {
		p.Stop()
	}

	// stop the consumers
	for _, c := range n.c {
		c.c.Stop()

		if len(n.lookupdAddrs) > 0 {
			// disconnect from all lookupd
			for _, addr := range n.lookupdAddrs {
				c.c.DisconnectFromNSQLookupd(addr)
			}
		} else {
			// disconnect from all nsq brokers
			for _, addr := range n.addrs {
				c.c.DisconnectFromNSQD(addr)
			}
		}
	}

	n.p = nil
	n.running = false
	return nil
}

func (n *nsqBroker) Publish(topic string, message *broker.Message, opts ...broker.PublishOption) error {
	p := n.p[rand.Intn(len(n.p))]

	options := broker.PublishOptions{}
	for _, o := range opts {
		o(&options)
	}

	var (
		doneChan chan *nsq.ProducerTransaction
		delay    time.Duration
	)
	if options.Context != nil {
		if v, ok := options.Context.Value(asyncPublishKey{}).(chan *nsq.ProducerTransaction); ok {
			doneChan = v
		}
		if v, ok := options.Context.Value(deferredPublishKey{}).(time.Duration); ok {
			delay = v
		}
	}

	b, err := n.opts.Codec.Marshal(message)
	if err != nil {
		return err
	}

	if doneChan != nil {
		if delay > 0 {
			return p.DeferredPublishAsync(topic, delay, b, doneChan)
		}
		return p.PublishAsync(topic, b, doneChan)
	} else {
		if delay > 0 {
			return p.DeferredPublish(topic, delay, b)
		}
		return p.Publish(topic, b)
	}
}

func (n *nsqBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	options := broker.SubscribeOptions{
		AutoAck: true,
	}

	for _, o := range opts {
		o(&options)
	}

	concurrency, maxInFlight := DefaultConcurrentHandlers, DefaultConcurrentHandlers
	if options.Context != nil {
		if v, ok := options.Context.Value(concurrentHandlerKey{}).(int); ok {
			maxInFlight, concurrency = v, v
		}
		if v, ok := options.Context.Value(maxInFlightKey{}).(int); ok {
			maxInFlight = v
		}
	}
	channel := options.Queue
	if len(channel) == 0 {
		channel = uuid.New().String() + "#ephemeral"
	}
	config := *n.config
	config.MaxInFlight = maxInFlight

	c, err := nsq.NewConsumer(topic, channel, &config)
	if err != nil {
		return nil, err
	}

	h := nsq.HandlerFunc(func(nm *nsq.Message) error {
		if !options.AutoAck {
			nm.DisableAutoResponse()
		}

		var m broker.Message

		if err := n.opts.Codec.Unmarshal(nm.Body, &m); err != nil {
			return err
		}

		p := &publication{topic: topic, m: &m}
		p.err = handler(p)
		return p.err
	})

	c.AddConcurrentHandlers(h, concurrency)

	if len(n.lookupdAddrs) > 0 {
		err = c.ConnectToNSQLookupds(n.lookupdAddrs)
	} else {
		err = c.ConnectToNSQDs(n.addrs)
	}
	if err != nil {
		return nil, err
	}

	sub := &subscriber{
		c:     c,
		opts:  options,
		topic: topic,
		h:     h,
		n:     concurrency,
	}

	n.c = append(n.c, sub)

	return sub, nil
}

func (n *nsqBroker) String() string {
	return "nsq"
}

func (p *publication) Topic() string {
	return p.topic
}

func (p *publication) Message() *broker.Message {
	return p.m
}

func (p *publication) Ack() error {
	p.nm.Finish()
	return nil
}

func (p *publication) Error() error {
	return p.err
}

func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.topic
}

func (s *subscriber) Unsubscribe() error {
	s.c.Stop()
	return nil
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.Options{
		// Default codec
		Codec: json.Marshaler{},
		// Default context
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}

	var addrs []string

	for _, addr := range options.Addrs {
		if len(addr) > 0 {
			addrs = append(addrs, addr)
		}
	}

	if len(addrs) == 0 {
		addrs = []string{"127.0.0.1:4150"}
	}

	n := &nsqBroker{
		addrs:  addrs,
		opts:   options,
		config: nsq.NewConfig(),
	}
	n.configure(n.opts.Context)

	return n
}
