package transport

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"

	maddr "go-micro.dev/v4/util/addr"
	mnet "go-micro.dev/v4/util/net"
)

type memorySocket struct {
	// True server mode, False client mode
	server bool
	// Client receiver of io.Pipe with gob
	crecv *gob.Decoder
	// Client sender of the io.Pipe with gob
	csend *gob.Encoder
	// Server receiver of the io.Pip with gob
	srecv *gob.Decoder
	// Server sender of the io.Pip with gob
	ssend *gob.Encoder
	// sock exit
	exit chan bool
	// listener exit
	lexit chan bool

	local  string
	remote string

	// for send/recv Timeout
	timeout time.Duration
	ctx     context.Context
}

type memoryClient struct {
	*memorySocket
	opts DialOptions
}

type memoryListener struct {
	addr  string
	exit  chan bool
	conn  chan *memorySocket
	lopts ListenOptions
	topts Options
	sync.RWMutex
	ctx context.Context
}

type memoryTransport struct {
	opts Options
	sync.RWMutex
	listeners map[string]*memoryListener
}

func (ms *memorySocket) Recv(m *Message) error {
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
		// connection closed
		return io.EOF
	case <-ms.lexit:
		// Server connection closed
		return io.EOF
	default:
		if ms.server {
			if err := ms.srecv.Decode(m); err != nil {
				return err
			}
		} else {
			if err := ms.crecv.Decode(m); err != nil {
				return err
			}
		}
	}

	return nil
}

func (ms *memorySocket) Local() string {
	return ms.local
}

func (ms *memorySocket) Remote() string {
	return ms.remote
}

func (ms *memorySocket) Send(m *Message) error {
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
		// connection closed
		return io.EOF
	case <-ms.lexit:
		// Server connection closed
		return io.EOF
	default:
		if ms.server {
			if err := ms.ssend.Encode(m); err != nil {
				return err
			}
		} else {
			if err := ms.csend.Encode(m); err != nil {
				return err
			}
		}
	}

	return nil
}

func (ms *memorySocket) Close() error {
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

func (m *memoryListener) Accept(fn func(Socket)) error {
	for {
		select {
		case <-m.exit:
			return nil
		case c := <-m.conn:
			go fn(&memorySocket{
				server:  true,
				lexit:   c.lexit,
				exit:    c.exit,
				ssend:   c.ssend,
				srecv:   c.srecv,
				local:   c.Remote(),
				remote:  c.Local(),
				timeout: m.topts.Timeout,
				ctx:     m.topts.Context,
			})
		}
	}
}

func (m *memoryTransport) Dial(addr string, opts ...DialOption) (Client, error) {
	m.RLock()
	defer m.RUnlock()

	listener, ok := m.listeners[addr]
	if !ok {
		return nil, errors.New("could not dial " + addr)
	}

	var options DialOptions
	for _, o := range opts {
		o(&options)
	}

	creader, swriter := io.Pipe()
	sreader, cwriter := io.Pipe()

	client := &memoryClient{
		&memorySocket{
			server: false,
			csend:  gob.NewEncoder(cwriter),
			crecv:  gob.NewDecoder(creader),
			ssend:  gob.NewEncoder(swriter),
			srecv:  gob.NewDecoder(sreader), exit: make(chan bool),
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

func (m *memoryTransport) Listen(addr string, opts ...ListenOption) (Listener, error) {
	m.Lock()
	defer m.Unlock()

	var options ListenOptions
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

func (m *memoryTransport) Init(opts ...Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}

func (m *memoryTransport) Options() Options {
	return m.opts
}

func (m *memoryTransport) String() string {
	return "memory"
}

func NewMemoryTransport(opts ...Option) Transport {
	var options Options

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
