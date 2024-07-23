package pool

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"go-micro.dev/v5/transport"
)

type pool struct {
	tr transport.Transport

	closeTimeout time.Duration
	conns        map[string][]*poolConn
	mu           sync.Mutex
	size         int
	ttl          time.Duration
}

type poolConn struct {
	transport.Client

	closeTimeout time.Duration
	created      time.Time
	id           string
}

func newPool(options Options) *pool {
	return &pool{
		size:         options.Size,
		tr:           options.Transport,
		ttl:          options.TTL,
		closeTimeout: options.CloseTimeout,
		conns:        make(map[string][]*poolConn),
	}
}

func (p *pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var err error

	for k, c := range p.conns {
		for _, conn := range c {
			if nerr := conn.close(); nerr != nil {
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
	p.mu.Lock()
	conns := p.conns[addr]

	// While we have conns check age and then return one
	// otherwise we'll create a new conn
	for len(conns) > 0 {
		conn := conns[len(conns)-1]
		conns = conns[:len(conns)-1]
		p.conns[addr] = conns

		// If conn is old kill it and move on
		if d := time.Since(conn.Created()); d > p.ttl {
			if err := conn.close(); err != nil {
				p.mu.Unlock()
				c, errConn := p.newConn(addr, opts)
				if errConn != nil {
					return nil, errConn
				}
				return c, err
			}

			continue
		}

		// We got a good conn, lets unlock and return it
		p.mu.Unlock()

		return conn, nil
	}

	p.mu.Unlock()

	return p.newConn(addr, opts)
}

func (p *pool) newConn(addr string, opts []transport.DialOption) (Conn, error) {
	// create new conn
	c, err := p.tr.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}

	return &poolConn{
		Client:       c,
		id:           uuid.New().String(),
		closeTimeout: p.closeTimeout,
		created:      time.Now(),
	}, nil
}

func (p *pool) Release(conn Conn, err error) error {
	// don't store the conn if it has errored
	if err != nil {
		return conn.(*poolConn).close()
	}

	// otherwise put it back for reuse
	p.mu.Lock()
	defer p.mu.Unlock()

	conns := p.conns[conn.Remote()]
	if len(conns) >= p.size {
		return conn.(*poolConn).close()
	}

	p.conns[conn.Remote()] = append(conns, conn.(*poolConn))

	return nil
}

func (p *poolConn) close() error {
	ch := make(chan error)
	go func() {
		defer close(ch)
		ch <- p.Client.Close()
	}()
	t := time.NewTimer(p.closeTimeout)
	var err error
	select {
	case <-t.C:
		err = errors.New("unable to close in time")
	case err = <-ch:
		t.Stop()
	}
	return err
}
