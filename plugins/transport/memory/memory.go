// Package memory is an in-memory transport
package memory

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/transport"
	maddr "github.com/micro/go-micro/v2/util/addr"
	mnet "github.com/micro/go-micro/v2/util/net"
)

type memorySocket struct {
	recv chan *transport.Message
	send chan *transport.Message
	// sock exit
	exit chan bool
	// listener exit
	lexit chan bool

	local  string
	remote string

	// for send/recv transport.Timeout
	timeout time.Duration
	ctx     context.Context
	sync.RWMutex
}

type memoryClient struct {
	*memorySocket
	opts transport.DialOptions
}

type memoryListener struct {
	addr  string
	exit  chan bool
	conn  chan *memorySocket
	lopts transport.ListenOptions
	topts transport.Options
	sync.RWMutex
	ctx context.Context
}

type memoryTransport struct {
	opts transport.Options
	sync.RWMutex
	listeners map[string]*memoryListener
}

func (ms *memorySocket) Recv(m *transport.Message) error {
	ms.RLock()
	defer ms.RUnlock()

	ctx := ms.ctx
	if ms.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ms.ctx, ms.timeout)
		defer cancel()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ms.exit:
		return errors.New("connection closed")
	case <-ms.lexit:
		return errors.New("server connection closed")
	case cm := <-ms.recv:
		*m = *cm
	}
	return nil
}

func (ms *memorySocket) Local() string {
	return ms.local
}

func (ms *memorySocket) Remote() string {
	return ms.remote
}

func (ms *memorySocket) Send(m *transport.Message) error {
	ms.RLock()
	defer ms.RUnlock()

	ctx := ms.ctx
	if ms.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ms.ctx, ms.timeout)
		defer cancel()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ms.exit:
		return errors.New("connection closed")
	case <-ms.lexit:
		return errors.New("server connection closed")
	case ms.send <- m:
	}
	return nil
}

func (ms *memorySocket) Close() error {
	ms.Lock()
	defer ms.Unlock()
	select {
	case <-ms.exit:
		return nil
	default:
		close(ms.exit)
	}
	return nil
}

func (m *memoryListener) Addr() string {
	return m.addr
}

func (m *memoryListener) Close() error {
	m.Lock()
	defer m.Unlock()
	select {
	case <-m.exit:
		return nil
	default:
		close(m.exit)
	}
	return nil
}

func (m *memoryListener) Accept(fn func(transport.Socket)) error {
	for {
		select {
		case <-m.exit:
			return nil
		case c := <-m.conn:
			go fn(&memorySocket{
				lexit:   c.lexit,
				exit:    c.exit,
				send:    c.recv,
				recv:    c.send,
				local:   c.Remote(),
				remote:  c.Local(),
				timeout: m.topts.Timeout,
				ctx:     m.topts.Context,
			})
		}
	}
}

func (m *memoryTransport) Dial(addr string, opts ...transport.DialOption) (transport.Client, error) {
	m.RLock()
	defer m.RUnlock()

	listener, ok := m.listeners[addr]
	if !ok {
		return nil, errors.New("could not dial " + addr)
	}

	var options transport.DialOptions
	for _, o := range opts {
		o(&options)
	}

	client := &memoryClient{
		&memorySocket{
			send:    make(chan *transport.Message),
			recv:    make(chan *transport.Message),
			exit:    make(chan bool),
			lexit:   listener.exit,
			local:   addr,
			remote:  addr,
			timeout: m.opts.Timeout,
			ctx:     m.opts.Context,
		},
		options,
	}

	// pseudo connect
	select {
	case <-listener.exit:
		return nil, errors.New("connection error")
	case listener.conn <- client.memorySocket:
	}

	return client, nil
}

func (m *memoryTransport) Listen(addr string, opts ...transport.ListenOption) (transport.Listener, error) {
	m.Lock()
	defer m.Unlock()

	var options transport.ListenOptions
	for _, o := range opts {
		o(&options)
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	addr, err = maddr.Extract(host)
	if err != nil {
		return nil, err
	}

	// if zero port then randomly assign one
	if len(port) > 0 && port == "0" {
		i := rand.Intn(20000)
		port = fmt.Sprintf("%d", 10000+i)
	}

	// set addr with port
	addr = mnet.HostPort(addr, port)

	if _, ok := m.listeners[addr]; ok {
		return nil, errors.New("already listening on " + addr)
	}

	listener := &memoryListener{
		lopts: options,
		topts: m.opts,
		addr:  addr,
		conn:  make(chan *memorySocket),
		exit:  make(chan bool),
		ctx:   m.opts.Context,
	}

	m.listeners[addr] = listener

	return listener, nil
}

func (m *memoryTransport) Init(opts ...transport.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}

func (m *memoryTransport) Options() transport.Options {
	return m.opts
}

func (m *memoryTransport) String() string {
	return "memory"
}

func NewTransport(opts ...transport.Option) transport.Transport {
	var options transport.Options

	rand.Seed(time.Now().UnixNano())

	for _, o := range opts {
		o(&options)
	}

	if options.Context == nil {
		options.Context = context.Background()
	}

	return &memoryTransport{
		opts:      options,
		listeners: make(map[string]*memoryListener),
	}
}
