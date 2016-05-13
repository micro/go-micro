package client

import (
	"sync"

	"github.com/micro/go-micro/transport"
)

type pool struct {
	tr transport.Transport

	sync.Mutex
	conns map[string][]*poolConn
}

type poolConn struct {
	transport.Client
}

var (
	maxIdleConn = 2
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
	conns, ok := p.conns[addr]
	// no free conn
	if !ok || len(conns) == 0 {
		p.Unlock()
		// create new conn
		c, err := tr.Dial(addr, opts...)
		if err != nil {
			return nil, err
		}
		return &poolConn{c}, nil
	}

	conn := conns[len(conns)-1]
	p.conns[addr] = conns[:len(conns)-1]
	p.Unlock()
	return conn, nil
}

func (p *pool) release(addr string, conn *poolConn, err error) {
	// don't store the conn
	if err != nil {
		conn.Client.Close()
		return
	}

	// otherwise put it back
	p.Lock()
	conns := p.conns[addr]
	if len(conns) >= maxIdleConn {
		conn.Client.Close()
		return
	}
	p.conns[addr] = append(conns, conn)
	p.Unlock()
}
