package stomp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	"github.com/go-stomp/stomp"
	"github.com/go-stomp/stomp/frame"
	"github.com/asim/go-micro/v3/broker"
	"github.com/asim/go-micro/v3/cmd"
)

type rbroker struct {
	opts      broker.Options
	stompConn *stomp.Conn
}

// init registers the STOMP broker
func init() {
	cmd.DefaultBrokers["stomp"] = NewBroker
}

// stompHeaderToMap converts STOMP header to broker friendly header
func stompHeaderToMap(h *frame.Header) map[string]string {
	m := map[string]string{}
	for i := 0; i < h.Len(); i++ {
		k, v := h.GetAt(i)
		m[k] = v
	}
	return m
}

// defaults sets 'sane' STOMP default
func (r *rbroker) defaults() {
	ConnectTimeout(30 * time.Second)(&r.opts)
	VirtualHost("/")(&r.opts)
}

func (r *rbroker) Options() broker.Options {
	if r.opts.Context == nil {
		r.opts.Context = context.Background()
	}
	return r.opts
}

func (r *rbroker) Address() string {
	if len(r.opts.Addrs) > 0 {
		return r.opts.Addrs[0]
	}
	return ""
}

func (r *rbroker) Connect() error {
	connectTimeOut := r.Options().Context.Value(connectTimeoutKey{}).(time.Duration)

	// Decode address
	url, err := url.Parse(r.Address())
	if err != nil {
		return err
	}

	// Make sure we are stomp
	if url.Scheme != "stomp" {
		return fmt.Errorf("Expected stomp:// protocol but was %s", url.Scheme)
	}

	// Retrieve user/pass if present
	stompOpts := []func(*stomp.Conn) error{}
	if url.User != nil && url.User.Username() != "" {
		password, _ := url.User.Password()
		stompOpts = append(stompOpts, stomp.ConnOpt.Login(url.User.Username(), password))
	}

	// Dial
	netConn, err := net.DialTimeout("tcp", url.Host, connectTimeOut)
	if err != nil {
		return fmt.Errorf("Failed to dial %s: %v", url.Host, err)
	}

	// Set connect options
	if auth, ok := r.Options().Context.Value(authKey{}).(*authRecord); ok && auth != nil {
		stompOpts = append(stompOpts, stomp.ConnOpt.Login(auth.username, auth.password))
	}
	if headers, ok := r.Options().Context.Value(connectHeaderKey{}).(map[string]string); ok && headers != nil {
		for k, v := range headers {
			stompOpts = append(stompOpts, stomp.ConnOpt.Header(k, v))
		}
	}
	if host, ok := r.Options().Context.Value(vHostKey{}).(string); ok && host != "" {
		log.Printf("Adding host: %s", host)
		stompOpts = append(stompOpts, stomp.ConnOpt.Host(host))
	}

	// STOMP Connect
	r.stompConn, err = stomp.Connect(netConn, stompOpts...)
	if err != nil {
		netConn.Close()
		return fmt.Errorf("Failed to connect to %s: %v", url.Host, err)
	}
	return nil
}

func (r *rbroker) Disconnect() error {
	return r.stompConn.Disconnect()
}

func (r *rbroker) Init(opts ...broker.Option) error {
	r.defaults()

	for _, o := range opts {
		o(&r.opts)
	}

	return nil
}

func (r *rbroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	if r.stompConn == nil {
		return errors.New("not connected")
	}

	// Set options
	stompOpt := make([]func(*frame.Frame) error, 0, len(msg.Header))
	for k, v := range msg.Header {
		stompOpt = append(stompOpt, stomp.SendOpt.Header(k, v))
	}

	bOpt := broker.PublishOptions{}
	for _, o := range opts {
		o(&bOpt)
	}
	if withReceipt, ok := r.Options().Context.Value(receiptKey{}).(bool); ok && withReceipt {
		stompOpt = append(stompOpt, stomp.SendOpt.Receipt)
	}
	if withoutContentLength, ok := r.Options().Context.Value(suppressContentLengthKey{}).(bool); ok && withoutContentLength {
		stompOpt = append(stompOpt, stomp.SendOpt.NoContentLength)
	}

	// Send
	if err := r.stompConn.Send(
		topic,
		"",
		msg.Body,
		stompOpt...); err != nil {
		return err
	}

	return nil
}

func (r *rbroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	var ackSuccess bool

	if r.stompConn == nil {
		return nil, errors.New("not connected")
	}

	// Set options
	stompOpt := make([]func(*frame.Frame) error, 0, len(opts))
	bOpt := broker.SubscribeOptions{
		AutoAck: true,
	}
	for _, o := range opts {
		o(&bOpt)
	}
	// Make sure context is setup
	if bOpt.Context == nil {
		bOpt.Context = context.Background()
	}

	ctx := bOpt.Context
	if subscribeContext, ok := ctx.Value(subscribeContextKey{}).(context.Context); ok && subscribeContext != nil {
		ctx = subscribeContext
	}

	if durableQueue, ok := ctx.Value(durableQueueKey{}).(bool); ok && durableQueue {
		stompOpt = append(stompOpt, stomp.SubscribeOpt.Header("persistent", "true"))
	}

	if headers, ok := ctx.Value(subscribeHeaderKey{}).(map[string]string); ok && len(headers) > 0 {
		for k, v := range headers {
			stompOpt = append(stompOpt, stomp.SubscribeOpt.Header(k, v))
		}
	}

	if bval, ok := ctx.Value(ackSuccessKey{}).(bool); ok && bval {
		bOpt.AutoAck = false
		ackSuccess = true
	}

	var ackMode stomp.AckMode
	if bOpt.AutoAck {
		ackMode = stomp.AckAuto
	} else {
		ackMode = stomp.AckClientIndividual
	}

	// Subscribe now
	sub, err := r.stompConn.Subscribe(topic, ackMode, stompOpt...)
	if err != nil {
		return nil, err
	}

	// Process messages
	go func() {
		for msg := range sub.C {
			go func(msg *stomp.Message) {
				// Transform message
				m := &broker.Message{
					Header: stompHeaderToMap(msg.Header),
					Body:   msg.Body,
				}
				p := &publication{msg: msg, m: m, topic: topic, broker: r}
				// Handle the publication
				p.err = handler(p)
				if p.err == nil && !bOpt.AutoAck && ackSuccess {
					msg.Conn.Ack(msg)
				}
			}(msg)
		}
	}()

	// Return subs
	return &subscriber{sub: sub, topic: topic, opts: bOpt}, nil
}

func (r *rbroker) String() string {
	return "stomp"
}

// NewBroker returns a STOMP broker
func NewBroker(opts ...broker.Option) broker.Broker {
	r := &rbroker{
		opts: broker.Options{
			Context: context.Background(),
		},
	}
	r.Init(opts...)
	return r
}
