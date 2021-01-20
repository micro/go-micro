// Package gocloud provides a pubsub broker for Go Cloud.
// Go Cloud offers cloud-agnostic interface for a variety of common cloud APIs.
// See https://github.com/google/go-cloud.
package gocloud

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/asim/go-micro/v3/broker"
	"github.com/asim/go-micro/v3/cmd"
	"github.com/streadway/amqp"
	"gocloud.dev/gcp"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/gcppubsub"
	"gocloud.dev/pubsub/mempubsub"
	"gocloud.dev/pubsub/rabbitpubsub"
)

func init() {
	cmd.DefaultBrokers["gocloud"] = NewBroker
}

type (
	topicOpener func(string) *pubsub.Topic
	subOpener   func(*pubsub.Topic, string) *pubsub.Subscription
)

type pubsubBroker struct {
	options   broker.Options
	openTopic topicOpener
	openSub   subOpener
	err       error

	mu     sync.Mutex
	topics map[string]*pubsub.Topic
	subs   map[string]*pubsub.Subscription
}

// NewBroker creates a new gocloud pubsubBroker.
// If the GCPProjectID option is set, Go Cloud uses its Google Cloud PubSub implementation.
// If the RabbitURL option is set, Go Cloud uses its RabbitMQ implementation.
// Otherwise, Go Cloud uses its in-memory implementation.
func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.Options{
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}
	var openTopic topicOpener
	var openSub subOpener
	var err error
	if projID, ok := options.Context.Value(gcpProjectIDKey{}).(gcp.ProjectID); ok {
		ts := options.Context.Value(gcpTokenSourceKey{}).(gcp.TokenSource)
		openTopic, openSub, err = setupGCP(options.Context, projID, ts)
	} else if rurl, ok := options.Context.Value(rabbitURLKey{}).(string); ok {
		openTopic, openSub, err = setupRabbit(options.Context, rurl)
	} else {
		openTopic, openSub, err = setupMem()
	}
	if err != nil {
		return &pubsubBroker{err: err}
	}
	return &pubsubBroker{
		options:   options,
		openTopic: openTopic,
		openSub:   openSub,
		topics:    map[string]*pubsub.Topic{},
		subs:      map[string]*pubsub.Subscription{},
	}
}

func setupGCP(ctx context.Context, projectID gcp.ProjectID, ts gcp.TokenSource) (topicOpener, subOpener, error) {
	conn, _, err := gcppubsub.Dial(ctx, ts) // ignore closer; no way to close
	if err != nil {
		return nil, nil, err
	}
	pc, err := gcppubsub.PublisherClient(ctx, conn)
	if err != nil {
		return nil, nil, err
	}
	sc, err := gcppubsub.SubscriberClient(ctx, conn)
	if err != nil {
		return nil, nil, err
	}
	return func(name string) *pubsub.Topic {
			return gcppubsub.OpenTopic(pc, projectID, name, nil)
		},
		func(t *pubsub.Topic, name string) *pubsub.Subscription {
			return gcppubsub.OpenSubscription(sc, projectID, name, nil)
		},
		nil
}

func setupRabbit(ctx context.Context, url string) (topicOpener, subOpener, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, err
	}
	return func(name string) *pubsub.Topic { return rabbitpubsub.OpenTopic(conn, name, nil) },
		func(_ *pubsub.Topic, name string) *pubsub.Subscription {
			return rabbitpubsub.OpenSubscription(conn, name, nil)
		},
		nil
}

func setupMem() (topicOpener, subOpener, error) {
	return func(string) *pubsub.Topic { return mempubsub.NewTopic() },
		func(t *pubsub.Topic, _ string) *pubsub.Subscription {
			return mempubsub.NewSubscription(t, 10*time.Second)
		},
		nil
}

func (b *pubsubBroker) String() string          { return "gocloud" }
func (b *pubsubBroker) Address() string         { return "" }
func (b *pubsubBroker) Connect() error          { return nil }
func (b *pubsubBroker) Disconnect() error       { return nil }
func (b *pubsubBroker) Options() broker.Options { return b.options }

func (b *pubsubBroker) Init(opts ...broker.Option) error {
	return errors.New("unimplemented; pass options to NewBroker instead")
}

// Publish opens the topic if it hasn't been already, then publishes the message.
func (b *pubsubBroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	if b.err != nil {
		return b.err
	}
	t := b.topic(topic)
	return t.Send(context.Background(), &pubsub.Message{Metadata: msg.Header, Body: msg.Body})
}

// Subscribe opens a subscription to the given topic and begins receiving messages and passing
// them to handler in a separate goroutine.
//
// The Queue SubscribeOption is required. Subscribe never creates new subscriptions on the backend (except
// for mempubsub, but you still need a queue name).
func (b *pubsubBroker) Subscribe(topic string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if b.err != nil {
		return nil, b.err
	}
	options := broker.SubscribeOptions{
		AutoAck: false,
		Context: b.options.Context,
	}
	for _, o := range opts {
		o(&options)
	}
	if options.Queue == "" {
		return nil, errors.New("gocloud.Subscribe: need Queue option")
	}
	t := b.topic(topic)
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.subs[options.Queue]
	if s == nil {
		// TODO(jba): how to configure ack deadline?
		s = b.openSub(t, options.Queue)
		b.subs[options.Queue] = s
	}
	// TODO(jba): how can we verify that an existing subscription's topic matches?
	ctx, cancel := context.WithCancel(context.Background())
	sub := &subscriber{
		options: options,
		topic:   topic,
		sub:     s,
		cancel:  cancel,
	}
	go sub.run(ctx, h)
	return sub, nil
}

// topic returns the topic with the given name if this broker has
// seen it before. Otherwise it opens a new topic.
func (b *pubsubBroker) topic(name string) *pubsub.Topic {
	b.mu.Lock()
	defer b.mu.Unlock()
	t := b.topics[name]
	if t == nil {
		t = b.openTopic(name)
		b.topics[name] = t
	}
	return t
}

// A pubsub subscriber that manages handling of messages.
type subscriber struct {
	options broker.SubscribeOptions
	topic   string
	sub     *pubsub.Subscription
	cancel  func()
}

func (s *subscriber) Options() broker.SubscribeOptions { return s.options }
func (s *subscriber) Topic() string                    { return s.topic }

func (s *subscriber) Unsubscribe() error {
	s.cancel()
	return s.sub.Shutdown(context.Background())
}

func (s *subscriber) run(ctx context.Context, h broker.Handler) {
	for {
		m, err := s.sub.Receive(ctx)
		if err != nil {
			log.Printf("Receive returned %v; stopping", err)
			break
		}
		p := &publication{
			msg: &broker.Message{
				Header: m.Metadata,
				Body:   m.Body,
			},
			topic: s.topic,
			ack:   m.Ack,
		}
		if err := h(p); err != nil {
			p.err = err
			log.Printf("handler returned %v; continuing", err)
			continue
		}
		if s.options.AutoAck {
			p.Ack()
		}
	}
}

// A single publication received by a handler.
type publication struct {
	msg   *broker.Message
	topic string
	ack   func()
	err   error
}

func (p *publication) Topic() string            { return p.topic }
func (p *publication) Message() *broker.Message { return p.msg }
func (p *publication) Ack() error               { p.ack(); return nil }
func (p *publication) Error() error             { return p.err }
