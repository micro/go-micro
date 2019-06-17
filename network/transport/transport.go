// Package transport implements the network as a transport interface
package transport

import (
	"context"
	"time"

	"github.com/micro/go-micro/network"
	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/util/backoff"
)

type networkKey struct{}

// Transport is a network transport
type Transport struct {
	Network network.Network
	options transport.Options
}

// Socket is a transport socket
type Socket struct {
	// The service
	Service string

	// Send via Network.Send(Message)
	Network network.Network

	// Remote/Local
	remote, local string

	// the first message if its a listener
	message *network.Message
}

// Listener is a transport listener
type Listener struct {
	// The local service
	Service string

	// The network
	Network network.Network
}

func (s *Socket) Local() string {
	return s.local
}

func (s *Socket) Remote() string {
	return s.remote
}

func (s *Socket) Close() error {
	// TODO: should it close the network?
	return s.Network.Close()
}

func (t *Transport) Init(opts ...transport.Option) error {
	for _, o := range opts {
		o(&t.options)
	}
	return nil
}

func (t *Transport) Options() transport.Options {
	return t.options
}

func (t *Transport) Dial(service string, opts ...transport.DialOption) (transport.Client, error) {
	// TODO: establish pseudo socket?
	return &Socket{
		Service: service,
		Network: t.Network,
		remote:  service,
		// TODO: local
		local: "local",
	}, nil
}

func (t *Transport) Listen(service string, opts ...transport.ListenOption) (transport.Listener, error) {
	// TODO specify connect id
	if err := t.Network.Connect("micro.mu"); err != nil {
		return nil, err
	}

	// advertise the service
	if err := t.Network.Advertise(service); err != nil {
		return nil, err
	}

	return &Listener{
		Service: service,
		Network: t.Network,
	}, nil
}

func (t *Transport) String() string {
	return "network"
}

func (s *Socket) Send(msg *transport.Message) error {
	// TODO: set routing headers?
	return s.Network.Send(&network.Message{
		Header: msg.Header,
		Body:   msg.Body,
	})
}

func (s *Socket) Recv(msg *transport.Message) error {
	if msg == nil {
		msg = new(transport.Message)
	}

	// return first message
	if s.message != nil {
		msg.Header = s.message.Header
		msg.Body = s.message.Body
		s.message = nil
		return nil
	}

	m, err := s.Network.Accept()
	if err != nil {
		return err
	}

	msg.Header = m.Header
	msg.Body = m.Body
	return nil
}

func (l *Listener) Addr() string {
	return l.Service
}

func (l *Listener) Close() error {
	return l.Network.Close()
}

func (l *Listener) Accept(fn func(transport.Socket)) error {
	var i int

	for {
		msg, err := l.Network.Accept()
		if err != nil {
			// increment error counter
			i++

			// break if lots of error
			if i > 3 {
				return err
			}

			// otherwise continue
			time.Sleep(backoff.Do(i))
			continue
		}

		// reset
		i = 0

		// execute in go routine
		go fn(&Socket{
			Service: l.Service,
			Network: l.Network,
			local:   l.Service,
			// TODO: remote
			remote:  "remote",
			message: msg,
		})
	}
}

// NewTransport returns a new network transport. It assumes the network is already connected
func NewTransport(opts ...transport.Option) transport.Transport {
	options := transport.Options{
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}

	net, ok := options.Context.Value(networkKey{}).(network.Network)
	if !ok {
		net = network.DefaultNetwork
	}

	return &Transport{
		options: options,
		Network: net,
	}
}

// WithNetwork passes in the network
func WithNetwork(n network.Network) transport.Option {
	return func(o *transport.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, networkKey{}, n)
	}
}
