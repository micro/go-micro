package mock

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/transport"
)

type mockSocket struct {
	recv chan *transport.Message
	send chan *transport.Message
	// sock exit
	exit chan bool
	// listener exit
	lexit chan bool
}

type mockClient struct {
	*mockSocket
	opts transport.DialOptions
}

type mockListener struct {
	addr string
	exit chan bool
	conn chan *mockSocket
	opts transport.ListenOptions
}

type mockTransport struct {
	opts transport.Options

	sync.Mutex
	listeners map[string]*mockListener
}

func (ms *mockSocket) Recv(m *transport.Message) error {
	select {
	case <-ms.exit:
		return errors.New("connection closed")
	case <-ms.lexit:
		return errors.New("server connection closed")
	case cm := <-ms.recv:
		*m = *cm
	}
	return nil
}

func (ms *mockSocket) Send(m *transport.Message) error {
	select {
	case <-ms.exit:
		return errors.New("connection closed")
	case <-ms.lexit:
		return errors.New("server connection closed")
	case ms.send <- m:
	}
	return nil
}

func (ms *mockSocket) Close() error {
	select {
	case <-ms.exit:
		return nil
	default:
		close(ms.exit)
	}
	return nil
}

func (m *mockListener) Addr() string {
	return m.addr
}

func (m *mockListener) Close() error {
	select {
	case <-m.exit:
		return nil
	default:
		close(m.exit)
	}
	return nil
}

func (m *mockListener) Accept(fn func(transport.Socket)) error {
	for {
		select {
		case <-m.exit:
			return nil
		case c := <-m.conn:
			go fn(&mockSocket{
				lexit: c.lexit,
				exit:  c.exit,
				send:  c.recv,
				recv:  c.send,
			})
		}
	}
}

func (m *mockTransport) Dial(addr string, opts ...transport.DialOption) (transport.Client, error) {
	m.Lock()
	defer m.Unlock()

	listener, ok := m.listeners[addr]
	if !ok {
		return nil, errors.New("could not dial " + addr)
	}

	var options transport.DialOptions
	for _, o := range opts {
		o(&options)
	}

	client := &mockClient{
		&mockSocket{
			send:  make(chan *transport.Message),
			recv:  make(chan *transport.Message),
			exit:  make(chan bool),
			lexit: listener.exit,
		},
		options,
	}

	// pseudo connect
	select {
	case <-listener.exit:
		return nil, errors.New("connection error")
	case listener.conn <- client.mockSocket:
	}

	return client, nil
}

func (m *mockTransport) Listen(addr string, opts ...transport.ListenOption) (transport.Listener, error) {
	m.Lock()
	defer m.Unlock()

	var options transport.ListenOptions
	for _, o := range opts {
		o(&options)
	}

	parts := strings.Split(addr, ":")

	// if zero port then randomly assign one
	if len(parts) > 1 && parts[len(parts)-1] == "0" {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		i := r.Intn(10000)
		// set addr with port
		addr = fmt.Sprintf("%s:%d", parts[:len(parts)-1], 10000+i)
	}

	if _, ok := m.listeners[addr]; ok {
		return nil, errors.New("already listening on " + addr)
	}

	listener := &mockListener{
		opts: options,
		addr: addr,
		conn: make(chan *mockSocket),
		exit: make(chan bool),
	}

	m.listeners[addr] = listener

	return listener, nil
}

func (m *mockTransport) Init(opts ...transport.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}

func (m *mockTransport) Options() transport.Options {
	return m.opts
}

func (m *mockTransport) String() string {
	return "mock"
}

func NewTransport(opts ...transport.Option) transport.Transport {
	var options transport.Options
	for _, o := range opts {
		o(&options)
	}

	return &mockTransport{
		opts:      options,
		listeners: make(map[string]*mockListener),
	}
}
