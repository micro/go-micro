// Package nats provides a NATS broker
package nats

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	natsp "github.com/nats-io/nats.go"
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/codec/json"
	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/registry"
)

type natsBroker struct {
	sync.Once
	sync.RWMutex

	// indicate if we're connected
	connected bool

	addrs []string
	conn  *natsp.Conn     // single connection (used when pool is disabled)
	pool  *connectionPool // connection pool (used when pooling is enabled)
	opts  broker.Options
	nopts natsp.Options

	// pool configuration
	poolSize        int
	poolIdleTimeout time.Duration

	// should we drain the connection
	drain   bool
	closeCh chan (error)
}

type subscriber struct {
	s    *natsp.Subscription
	opts broker.SubscribeOptions
}

type publication struct {
	t   string
	err error
	m   *broker.Message
}

func (p *publication) Topic() string {
	return p.t
}

func (p *publication) Message() *broker.Message {
	return p.m
}

func (p *publication) Ack() error {
	// nats does not support acking
	return nil
}

func (p *publication) Error() error {
	return p.err
}

func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.s.Subject
}

func (s *subscriber) Unsubscribe() error {
	return s.s.Unsubscribe()
}

func (n *natsBroker) Address() string {
	if n.conn != nil && n.conn.IsConnected() {
		return n.conn.ConnectedUrl()
	}

	if len(n.addrs) > 0 {
		return n.addrs[0]
	}

	return ""
}

func (n *natsBroker) setAddrs(addrs []string) []string {
	//nolint:prealloc
	var cAddrs []string
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
		cAddrs = []string{natsp.DefaultURL}
	}
	return cAddrs
}

func (n *natsBroker) Connect() error {
	n.Lock()
	defer n.Unlock()

	if n.connected {
		return nil
	}

	// Check if we should use connection pooling
	if n.poolSize > 1 {
		// Initialize connection pool
		factory := func() (*natsp.Conn, error) {
			opts := n.nopts
			opts.Servers = n.addrs
			opts.Secure = n.opts.Secure
			opts.TLSConfig = n.opts.TLSConfig

			// secure might not be set
			if n.opts.TLSConfig != nil {
				opts.Secure = true
			}

			return opts.Connect()
		}

		pool, err := newConnectionPool(n.poolSize, factory)
		if err != nil {
			return err
		}

		// Set idle timeout if configured
		if n.poolIdleTimeout > 0 {
			pool.idleTimeout = n.poolIdleTimeout
		}

		n.pool = pool
		n.connected = true
		return nil
	}

	// Single connection mode (original behavior)
	status := natsp.CLOSED
	if n.conn != nil {
		status = n.conn.Status()
	}

	switch status {
	case natsp.CONNECTED, natsp.RECONNECTING, natsp.CONNECTING:
		n.connected = true
		return nil
	default: // DISCONNECTED or CLOSED or DRAINING
		opts := n.nopts
		opts.Servers = n.addrs
		opts.Secure = n.opts.Secure
		opts.TLSConfig = n.opts.TLSConfig

		// secure might not be set
		if n.opts.TLSConfig != nil {
			opts.Secure = true
		}

		c, err := opts.Connect()
		if err != nil {
			return err
		}
		n.conn = c
		n.connected = true
		return nil
	}
}

func (n *natsBroker) Disconnect() error {
	n.Lock()
	defer n.Unlock()

	// Close connection pool if it exists
	if n.pool != nil {
		if err := n.pool.Close(); err != nil {
			n.opts.Logger.Log(logger.ErrorLevel, "error closing connection pool:", err)
		}
		n.pool = nil
	}

	// Close single connection if it exists
	if n.conn != nil {
		// drain the connection if specified
		if n.drain {
			n.conn.Drain()
			n.closeCh <- nil
		}

		// close the client connection
		n.conn.Close()
		n.conn = nil
	}

	// set not connected
	n.connected = false

	return nil
}

func (n *natsBroker) Init(opts ...broker.Option) error {
	n.setOption(opts...)
	return nil
}

func (n *natsBroker) Options() broker.Options {
	return n.opts
}

func (n *natsBroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	n.RLock()
	defer n.RUnlock()

	b, err := n.opts.Codec.Marshal(msg)
	if err != nil {
		return err
	}

	// Use connection pool if enabled
	if n.pool != nil {
		poolConn, err := n.pool.Get()
		if err != nil {
			return err
		}
		defer n.pool.Put(poolConn)

		conn := poolConn.Conn()
		if conn == nil {
			return errors.New("invalid connection from pool")
		}
		return conn.Publish(topic, b)
	}

	// Use single connection (original behavior)
	if n.conn == nil {
		return errors.New("not connected")
	}

	return n.conn.Publish(topic, b)
}

func (n *natsBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	n.RLock()
	hasConnection := n.conn != nil || n.pool != nil
	n.RUnlock()

	if !hasConnection {
		return nil, errors.New("not connected")
	}

	opt := broker.SubscribeOptions{
		AutoAck: true,
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&opt)
	}

	fn := func(msg *natsp.Msg) {
		var m broker.Message
		pub := &publication{t: msg.Subject}
		eh := n.opts.ErrorHandler
		err := n.opts.Codec.Unmarshal(msg.Data, &m)
		pub.err = err
		pub.m = &m
		if err != nil {
			m.Body = msg.Data
			n.opts.Logger.Log(logger.ErrorLevel, err)
			if eh != nil {
				eh(pub)
			}
			return
		}
		if err := handler(pub); err != nil {
			pub.err = err
			n.opts.Logger.Log(logger.ErrorLevel, err)
			if eh != nil {
				eh(pub)
			}
		}
	}

	var sub *natsp.Subscription
	var err error

	// Use connection pool if enabled
	if n.pool != nil {
		poolConn, err := n.pool.Get()
		if err != nil {
			return nil, err
		}

		conn := poolConn.Conn()
		if conn == nil {
			n.pool.Put(poolConn)
			return nil, errors.New("invalid connection from pool")
		}

		if len(opt.Queue) > 0 {
			sub, err = conn.QueueSubscribe(topic, opt.Queue, fn)
		} else {
			sub, err = conn.Subscribe(topic, fn)
		}

		if err != nil {
			n.pool.Put(poolConn)
			return nil, err
		}

		// Return connection to pool after subscription is created
		// The subscription keeps the connection alive
		n.pool.Put(poolConn)

		return &subscriber{s: sub, opts: opt}, nil
	}

	// Use single connection (original behavior)
	n.RLock()
	if len(opt.Queue) > 0 {
		sub, err = n.conn.QueueSubscribe(topic, opt.Queue, fn)
	} else {
		sub, err = n.conn.Subscribe(topic, fn)
	}
	n.RUnlock()
	if err != nil {
		return nil, err
	}
	return &subscriber{s: sub, opts: opt}, nil
}

func (n *natsBroker) String() string {
	return "nats"
}

func (n *natsBroker) setOption(opts ...broker.Option) {
	for _, o := range opts {
		o(&n.opts)
	}

	n.Once.Do(func() {
		n.nopts = natsp.GetDefaultOptions()
		n.poolSize = 1 // Default to single connection (no pooling)
		n.poolIdleTimeout = 5 * time.Minute
	})

	if nopts, ok := n.opts.Context.Value(optionsKey{}).(natsp.Options); ok {
		n.nopts = nopts
	}

	// Set pool size if configured
	if poolSize, ok := n.opts.Context.Value(poolSizeKey{}).(int); ok && poolSize > 0 {
		n.poolSize = poolSize
	}

	// Set pool idle timeout if configured
	if idleTimeout, ok := n.opts.Context.Value(poolIdleTimeoutKey{}).(time.Duration); ok {
		n.poolIdleTimeout = idleTimeout
	}

	// broker.Options have higher priority than nats.Options
	// only if Addrs, Secure or TLSConfig were not set through a broker.Option
	// we read them from nats.Option
	if len(n.opts.Addrs) == 0 {
		n.opts.Addrs = n.nopts.Servers
	}

	if !n.opts.Secure {
		n.opts.Secure = n.nopts.Secure
	}

	if n.opts.TLSConfig == nil {
		n.opts.TLSConfig = n.nopts.TLSConfig
	}
	n.addrs = n.setAddrs(n.opts.Addrs)

	if n.opts.Context.Value(drainConnectionKey{}) != nil {
		n.drain = true
		n.closeCh = make(chan error)
		n.nopts.ClosedCB = n.onClose
		n.nopts.AsyncErrorCB = n.onAsyncError
		n.nopts.DisconnectedErrCB = n.onDisconnectedError
	}
}

func (n *natsBroker) onClose(conn *natsp.Conn) {
	n.closeCh <- nil
}

func (n *natsBroker) onAsyncError(conn *natsp.Conn, sub *natsp.Subscription, err error) {
	// There are kinds of different async error nats might callback, but we are interested
	// in ErrDrainTimeout only here.
	if err == natsp.ErrDrainTimeout {
		n.closeCh <- err
	}
}

func (n *natsBroker) onDisconnectedError(conn *natsp.Conn, err error) {
	n.closeCh <- err
}

func NewNatsBroker(opts ...broker.Option) broker.Broker {
	options := broker.Options{
		// Default codec
		Codec:    json.Marshaler{},
		Context:  context.Background(),
		Registry: registry.DefaultRegistry,
		Logger:   logger.DefaultLogger,
	}

	n := &natsBroker{
		opts: options,
	}
	n.setOption(opts...)

	return n
}
