package grpc

import (
	"sync"
	"time"

	"google.golang.org/grpc"
)

type pool struct {
	size int
	ttl  int64

	sync.Mutex
	conns map[string][]*poolConn
}

type poolConn struct {
	*grpc.ClientConn
	created int64
}

func newPool(size int, ttl time.Duration) *pool {
	return &pool{
		size:  size,
		ttl:   int64(ttl.Seconds()),
		conns: make(map[string][]*poolConn),
	}
}

func (p *pool) getConn(addr string, opts ...grpc.DialOption) (*poolConn, error) {
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
		if d := now - conn.created; d > p.ttl {
			conn.ClientConn.Close()
			continue
		}

		// we got a good conn, lets unlock and return it
		p.Unlock()

		return conn, nil
	}

	p.Unlock()

	// create new conn
	cc, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}

	return &poolConn{cc, time.Now().Unix()}, nil
}

func (p *pool) release(addr string, conn *poolConn, err error) {
	// don't store the conn if it has errored
	if err != nil {
		conn.ClientConn.Close()
		return
	}

	// otherwise put it back for reuse
	p.Lock()
	conns := p.conns[addr]
	if len(conns) >= p.size {
		p.Unlock()
		conn.ClientConn.Close()
		return
	}
	p.conns[addr] = append(conns, conn)
	p.Unlock()
}
