package pool

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"go-micro.dev/v4/transport"
)

type pool struct {
	size int
	ttl  time.Duration
	tr   transport.Transport

	sync.Mutex
	conns map[string][]*poolConn
}

type poolConn struct {
	transport.Client
	id      string
	created time.Time
}

func newPool(options Options) *pool {
	return &pool{
		size:  options.Size,
		tr:    options.Transport,
		ttl:   options.TTL,
		conns: make(map[string][]*poolConn),
	}
}

func (p *pool) Close() error {
	p.Lock()
	defer p.Unlock()

	var err error

	for k, c := range p.conns {
		for _, conn := range c {
			if nerr := conn.Client.Close(); nerr != nil {
				err = nerr
			}
		}

		delete(p.conns, k)
	}

	return err
}

// NoOp the Close since we manage it.
func (p *poolConn) Close() error {
	return nil
}

func (p *poolConn) Id() string {
	return p.id
}

func (p *poolConn) Created() time.Time {
	return p.created
}

func (p *pool) Get(addr string, opts ...transport.DialOption) (Conn, error) {
	p.Lock()
	conns := p.conns[addr]

	// While we have conns check age and then return one
	// otherwise we'll create a new conn
	for len(conns) > 0 {
		conn := conns[len(conns)-1]
		conns = conns[:len(conns)-1]
		p.conns[addr] = conns

		// If conn is old kill it and move on
		if d := time.Since(conn.Created()); d > p.ttl {
			if err := conn.Client.Close(); err != nil {
				p.Unlock()
				return nil, err
			}

			continue
		}

		// We got a good conn, lets unlock and return it
		p.Unlock()

		return conn, nil
	}

	p.Unlock()

	// create new conn
	c, err := p.tr.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}

	return &poolConn{
		Client:  c,
		id:      uuid.New().String(),
		created: time.Now(),
	}, nil
}

func (p *pool) Release(conn Conn, err error) error {
	// don't store the conn if it has errored
	if err != nil {
		return conn.(*poolConn).Client.Close()
	}

	// otherwise put it back for reuse
	p.Lock()
	defer p.Unlock()

	conns := p.conns[conn.Remote()]
	if len(conns) >= p.size {
		return conn.(*poolConn).Client.Close()
	}

	p.conns[conn.Remote()] = append(conns, conn.(*poolConn))

	return nil
}
