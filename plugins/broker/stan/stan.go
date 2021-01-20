// Package stan provides a NATS Streaming broker
package stan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/asim/go-micro/v3/broker"
	"github.com/asim/go-micro/v3/codec/json"
	"github.com/asim/go-micro/v3/cmd"
	log "github.com/asim/go-micro/v3/logger"
	stan "github.com/nats-io/stan.go"
)

type stanBroker struct {
	sync.RWMutex
	addrs          []string
	conn           stan.Conn
	opts           broker.Options
	sopts          stan.Options
	nopts          []stan.Option
	clusterID      string
	clientID       string
	connectTimeout time.Duration
	connectRetry   bool
	done           chan struct{}
	ctx            context.Context
}

type subscriber struct {
	t    string
	s    stan.Subscription
	dq   bool
	opts broker.SubscribeOptions
}

type publication struct {
	t   string
	msg *stan.Msg
	m   *broker.Message
	err error
}

func init() {
	cmd.DefaultBrokers["stan"] = NewBroker
}

func (n *publication) Topic() string {
	return n.t
}

func (n *publication) Message() *broker.Message {
	return n.m
}

func (n *publication) Ack() error {
	return n.msg.Ack()
}

func (n *publication) Error() error {
	return n.err
}

func (n *subscriber) Options() broker.SubscribeOptions {
	return n.opts
}

func (n *subscriber) Topic() string {
	return n.t
}

func (n *subscriber) Unsubscribe() error {
	if n.s == nil {
		return nil
	}
	// go-micro server Unsubscribe can't handle durable queues, so close as stan suggested
	// from nats streaming readme:
	// When a client disconnects, the streaming server is not notified, hence the importance of calling Close()
	if !n.dq {
		err := n.s.Unsubscribe()
		if err != nil {
			return err
		}
	}
	return n.Close()
}

func (n *subscriber) Close() error {
	if n.s != nil {
		return n.s.Close()
	}
	return nil
}

func (n *stanBroker) Address() string {
	// stan does not support connected server info
	if len(n.addrs) > 0 {
		return n.addrs[0]
	}

	return ""
}

func setAddrs(addrs []string) []string {
	cAddrs := make([]string, 0, len(addrs))
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
		cAddrs = []string{stan.DefaultNatsURL}
	}
	return cAddrs
}

func (n *stanBroker) reconnectCB(c stan.Conn, err error) {
	if n.connectRetry {
		if err := n.connect(); err != nil {
			log.Error(err)
		}
	}
}

func (n *stanBroker) connect() error {
	timeout := make(<-chan time.Time)

	n.RLock()
	if n.connectTimeout > 0 {
		timeout = time.After(n.connectTimeout)
	}
	clusterID := n.clusterID
	clientID := n.clientID
	nopts := n.nopts
	n.RUnlock()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	fn := func() error {
		c, err := stan.Connect(clusterID, clientID, nopts...)
		if err == nil {
			n.Lock()
			n.conn = c
			n.Unlock()
		}
		return err
	}

	// don't wait for first try
	if err := fn(); err == nil {
		return nil
	}

	n.RLock()
	done := n.done
	n.RUnlock()

	// wait loop
	for {
		select {
		// context closed
		case <-n.opts.Context.Done():
			return nil
		// call close, don't wait anymore
		case <-done:
			return nil
		//  in case of timeout fail with a timeout error
		case <-timeout:
			return fmt.Errorf("[stan]: timeout connect to %v", n.addrs)
		// got a tick, try to connect
		case <-ticker.C:
			err := fn()
			if err == nil {
				log.Infof("[stan]: successeful connected to %v", n.addrs)
				return nil
			}
			log.Errorf("[stan]: failed to connect %v: %v\n", n.addrs, err)
		}
	}

	return nil
}

func (n *stanBroker) Connect() error {
	n.RLock()
	if n.conn != nil {
		n.RUnlock()
		return nil
	}
	n.RUnlock()

	clusterID, ok := n.opts.Context.Value(clusterIDKey{}).(string)
	if !ok || len(clusterID) == 0 {
		return errors.New("must specify ClusterID Option")
	}

	clientID, ok := n.opts.Context.Value(clientIDKey{}).(string)
	if !ok || len(clientID) == 0 {
		clientID = uuid.New().String()
	}

	n.Lock()
	if v, ok := n.opts.Context.Value(connectRetryKey{}).(bool); ok && v {
		n.connectRetry = true
	}

	if td, ok := n.opts.Context.Value(connectTimeoutKey{}).(time.Duration); ok {
		n.connectTimeout = td
	}

	if n.sopts.ConnectionLostCB != nil && n.connectRetry {
		n.Unlock()
		return errors.New("impossible to use custom ConnectionLostCB and ConnectRetry(true)")
	}

	nopts := []stan.Option{
		stan.NatsURL(n.sopts.NatsURL),
		stan.NatsConn(n.sopts.NatsConn),
		stan.ConnectWait(n.sopts.ConnectTimeout),
		stan.PubAckWait(n.sopts.AckTimeout),
		stan.MaxPubAcksInflight(n.sopts.MaxPubAcksInflight),
		stan.Pings(n.sopts.PingInterval, n.sopts.PingMaxOut),
	}

	if n.connectRetry {
		nopts = append(nopts, stan.SetConnectionLostHandler(n.reconnectCB))
	}

	nopts = append(nopts, stan.NatsURL(strings.Join(n.addrs, ",")))

	n.nopts = nopts
	n.clusterID = clusterID
	n.clientID = clientID
	n.Unlock()

	return n.connect()
}

func (n *stanBroker) Disconnect() error {
	var err error

	n.Lock()
	defer n.Unlock()

	if n.done != nil {
		close(n.done)
		n.done = nil
	}
	if n.conn != nil {
		err = n.conn.Close()
	}
	return err
}

func (n *stanBroker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&n.opts)
	}
	n.addrs = setAddrs(n.opts.Addrs)
	return nil
}

func (n *stanBroker) Options() broker.Options {
	return n.opts
}

func (n *stanBroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	b, err := n.opts.Codec.Marshal(msg)
	if err != nil {
		return err
	}
	n.RLock()
	defer n.RUnlock()
	return n.conn.Publish(topic, b)
}

func (n *stanBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	n.RLock()
	if n.conn == nil {
		n.RUnlock()
		return nil, errors.New("not connected")
	}
	n.RUnlock()

	var ackSuccess bool

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

	var stanOpts []stan.SubscriptionOption
	if !opt.AutoAck {
		stanOpts = append(stanOpts, stan.SetManualAckMode())
	}

	if subOpts, ok := ctx.Value(subscribeOptionKey{}).([]stan.SubscriptionOption); ok && len(subOpts) > 0 {
		stanOpts = append(stanOpts, subOpts...)
	}

	if bval, ok := ctx.Value(ackSuccessKey{}).(bool); ok && bval {
		stanOpts = append(stanOpts, stan.SetManualAckMode())
		ackSuccess = true
	}

	bopts := stan.DefaultSubscriptionOptions
	for _, bopt := range stanOpts {
		if err := bopt(&bopts); err != nil {
			return nil, err
		}
	}

	opt.AutoAck = !bopts.ManualAcks

	if dn, ok := n.opts.Context.Value(durableKey{}).(string); ok && len(dn) > 0 {
		stanOpts = append(stanOpts, stan.DurableName(dn))
		bopts.DurableName = dn
	}

	fn := func(msg *stan.Msg) {
		var m broker.Message
		p := &publication{m: &m, msg: msg, t: msg.Subject}

		// unmarshal message
		if err := n.opts.Codec.Unmarshal(msg.Data, &m); err != nil {
			p.err = err
			p.m.Body = msg.Data
			return
		}
		// execute the handler
		p.err = handler(p)
		// if there's no error and success auto ack is enabled ack it
		if p.err == nil && ackSuccess {
			msg.Ack()
		}
	}

	var sub stan.Subscription
	var err error

	n.RLock()
	if len(opt.Queue) > 0 {
		sub, err = n.conn.QueueSubscribe(topic, opt.Queue, fn, stanOpts...)
	} else {
		sub, err = n.conn.Subscribe(topic, fn, stanOpts...)
	}
	n.RUnlock()
	if err != nil {
		return nil, err
	}
	return &subscriber{dq: len(bopts.DurableName) > 0, s: sub, opts: opt, t: topic}, nil
}

func (n *stanBroker) String() string {
	return "stan"
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.Options{
		// Default codec
		Codec:   json.Marshaler{},
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}

	stanOpts := stan.GetDefaultOptions()
	if n, ok := options.Context.Value(optionsKey{}).(stan.Options); ok {
		stanOpts = n
	}

	if len(options.Addrs) == 0 {
		options.Addrs = strings.Split(stanOpts.NatsURL, ",")
	}

	nb := &stanBroker{
		done:  make(chan struct{}),
		opts:  options,
		sopts: stanOpts,
		addrs: setAddrs(options.Addrs),
	}

	return nb
}
