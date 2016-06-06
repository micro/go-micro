package client

import (
	"sync"
	"time"

	"github.com/micro/go-micro/transport"
)

type pool struct {
	tr transport.Transport

	sync.Mutex
	conns map[string][]*poolConn
}

type poolConn struct {
	transport.Client
	created int64
}

var (
	// only hold on to this many conns
	maxIdleConn = 2
	// only hold on to the conn for this period
	maxLifeTime = int64(60)
)

func newPool() *pool {
	return &pool{
		conns: make(map[string][]*poolConn),
	}
}

// NoOp the Close since we manage it
func (p *poolConn) Close() error {
	return nil
}

func (p *pool) getConn(addr string, tr transport.Transport, opts ...transport.DialOption) (*poolConn, error) {
	p.Lock()
	conns := p.conns[addr]
	now := time.Now().Unix()

	// while we have conns check age and then return one
	// otherwise we'll create a new conn
	for len(conns) > 0 {
		conn := conns[len(conns)-1]
		conns = conns[:len(conns)-1]
		p.conns[addr] = conns

		// if conn is old kill it and move on
		if d := now - conn.created; d > maxLifeTime {
			conn.Client.Close()
			continue
		}

		// we got a good conn, lets unlock and return it
		p.Unlock()

		return conn, nil
	}

	p.Unlock()

	// create new conn
	c, err := tr.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	return &poolConn{c, time.Now().Unix()}, nil
}

func (p *pool) release(addr string, conn *poolConn, err error) {
	// don't store the conn if it has errored
	if err != nil {
		conn.Client.Close()
		return
	}

	// otherwise put it back for reuse
	p.Lock()
	conns := p.conns[addr]
	if len(conns) >= maxIdleConn {
		p.Unlock()
		conn.Client.Close()
		return
	}
	p.conns[addr] = append(conns, conn)
	p.Unlock()
}
