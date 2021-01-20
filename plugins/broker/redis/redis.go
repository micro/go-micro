// Package redis provides a Redis broker
package redis

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/asim/go-micro/v3/broker"
	"github.com/asim/go-micro/v3/codec"
	"github.com/asim/go-micro/v3/codec/json"
	"github.com/asim/go-micro/v3/cmd"
)

func init() {
	cmd.DefaultBrokers["redis"] = NewBroker
}

// publication is an internal publication for the Redis broker.
type publication struct {
	topic   string
	message *broker.Message
	err     error
}

// Topic returns the topic this publication applies to.
func (p *publication) Topic() string {
	return p.topic
}

// Message returns the broker message of the publication.
func (p *publication) Message() *broker.Message {
	return p.message
}

// Ack sends an acknowledgement to the broker. However this is not supported
// is Redis and therefore this is a no-op.
func (p *publication) Ack() error {
	return nil
}

func (p *publication) Error() error {
	return p.err
}

// subscriber proxies and handles Redis messages as broker publications.
type subscriber struct {
	codec  codec.Marshaler
	conn   *redis.PubSubConn
	topic  string
	handle broker.Handler
	opts   broker.SubscribeOptions
}

// recv loops to receive new messages from Redis and handle them
// as publications.
func (s *subscriber) recv() {
	// Close the connection once the subscriber stops receiving.
	defer s.conn.Close()

	for {
		switch x := s.conn.Receive().(type) {
		case redis.Message:
			var m broker.Message

			// Handle error? Only a log would be necessary since this type
			// of issue cannot be fixed.
			if err := s.codec.Unmarshal(x.Data, &m); err != nil {
				break
			}

			p := publication{
				topic:   x.Channel,
				message: &m,
			}

			// Handle error? Retry?
			if p.err = s.handle(&p); p.err != nil {
				break
			}

			// Added for posterity, however Ack is a no-op.
			if s.opts.AutoAck {
				if err := p.Ack(); err != nil {
					break
				}
			}

		case redis.Subscription:
			if x.Count == 0 {
				return
			}

		case error:
			return
		}
	}
}

// Options returns the subscriber options.
func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

// Topic returns the topic of the subscriber.
func (s *subscriber) Topic() string {
	return s.topic
}

// Unsubscribe unsubscribes the subscriber and frees the connection.
func (s *subscriber) Unsubscribe() error {
	return s.conn.Unsubscribe()
}

// broker implementation for Redis.
type redisBroker struct {
	addr  string
	pool  *redis.Pool
	opts  broker.Options
	bopts *brokerOptions
}

// String returns the name of the broker implementation.
func (b *redisBroker) String() string {
	return "redis"
}

// Options returns the options defined for the broker.
func (b *redisBroker) Options() broker.Options {
	return b.opts
}

// Address returns the address the broker will use to create new connections.
// This will be set only after Connect is called.
func (b *redisBroker) Address() string {
	return b.addr
}

// Init sets or overrides broker options.
func (b *redisBroker) Init(opts ...broker.Option) error {
	if b.pool != nil {
		return errors.New("redis: cannot init while connected")
	}

	for _, o := range opts {
		o(&b.opts)
	}

	return nil
}

// Connect establishes a connection to Redis which provides the
// pub/sub implementation.
func (b *redisBroker) Connect() error {
	if b.pool != nil {
		return nil
	}

	var addr string

	if len(b.opts.Addrs) == 0 || b.opts.Addrs[0] == "" {
		addr = "redis://127.0.0.1:6379"
	} else {
		addr = b.opts.Addrs[0]

		if !strings.HasPrefix("redis://", addr) {
			addr = "redis://" + addr
		}
	}

	b.addr = addr

	b.pool = &redis.Pool{
		MaxIdle:     b.bopts.maxIdle,
		MaxActive:   b.bopts.maxActive,
		IdleTimeout: b.bopts.idleTimeout,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(
				b.addr,
				redis.DialConnectTimeout(b.bopts.connectTimeout),
				redis.DialReadTimeout(b.bopts.readTimeout),
				redis.DialWriteTimeout(b.bopts.writeTimeout),
			)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return nil
}

// Disconnect closes the connection pool.
func (b *redisBroker) Disconnect() error {
	err := b.pool.Close()
	b.pool = nil
	b.addr = ""
	return err
}

// Publish publishes a message.
func (b *redisBroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	v, err := b.opts.Codec.Marshal(msg)
	if err != nil {
		return err
	}

	conn := b.pool.Get()
	_, err = redis.Int(conn.Do("PUBLISH", topic, v))
	conn.Close()

	return err
}

// Subscribe returns a subscriber for the topic and handler.
func (b *redisBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	var options broker.SubscribeOptions
	for _, o := range opts {
		o(&options)
	}

	s := subscriber{
		codec:  b.opts.Codec,
		conn:   &redis.PubSubConn{Conn: b.pool.Get()},
		topic:  topic,
		handle: handler,
		opts:   options,
	}

	// Run the receiver routine.
	go s.recv()

	if err := s.conn.Subscribe(s.topic); err != nil {
		return nil, err
	}

	return &s, nil
}

// NewBroker returns a new broker implemented using the Redis pub/sub
// protocol. The connection address may be a fully qualified IANA address such
// as: redis://user:secret@localhost:6379/0?foo=bar&qux=baz
func NewBroker(opts ...broker.Option) broker.Broker {
	// Default options.
	bopts := &brokerOptions{
		maxIdle:        DefaultMaxIdle,
		maxActive:      DefaultMaxActive,
		idleTimeout:    DefaultIdleTimeout,
		connectTimeout: DefaultConnectTimeout,
		readTimeout:    DefaultReadTimeout,
		writeTimeout:   DefaultWriteTimeout,
	}

	// Initialize with empty broker options.
	options := broker.Options{
		Codec:   json.Marshaler{},
		Context: context.WithValue(context.Background(), optionsKey, bopts),
	}

	for _, o := range opts {
		o(&options)
	}

	return &redisBroker{
		opts:  options,
		bopts: bopts,
	}
}
