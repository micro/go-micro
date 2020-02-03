package pool

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v2/transport"
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
	for k, c := range p.conns {
		for _, conn := range c {
			conn.Client.Close()
		}
		delete(p.conns, k)
	}
	p.Unlock()
	return nil
}

// NoOp the Close since we manage it
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

	// while we have conns check age and then return one
	// otherwise we'll create a new conn
	for len(conns) > 0 {
		conn := conns[len(conns)-1]
		conns = conns[:len(conns)-1]
		p.conns[addr] = conns

		// if conn is old kill it and move on
		if d := time.Since(conn.Created()); d > p.ttl {
			conn.Client.Close()
			continue
		}

		// we got a good conn, lets unlock and return it
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
	conns := p.conns[conn.Remote()]
	if len(conns) >= p.size {
		p.Unlock()
		return conn.(*poolConn).Client.Close()
	}
	p.conns[conn.Remote()] = append(conns, conn.(*poolConn))
	p.Unlock()

	return nil
}
