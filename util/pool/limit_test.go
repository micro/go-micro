package pool

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/transport"
)

func TestLimitDefault(t *testing.T) {
	tr, addr, closeListener := newTestTransport(remoteHandler)
	defer closeListener() // nolint
	pool := newLimitPool(Options{Transport: tr})
	defer pool.Close()

	require.Equal(t, 0, pool.CountIdleConnsTesting(addr))

	c, err := pool.Get(addr)
	require.NoError(t, err)

	msg := &transport.Message{
		Body: []byte("hello world"),
	}
	require.NoError(t, c.Send(msg))

	require.NoError(t, c.Recv(&transport.Message{}))
	require.NoError(t, pool.Release(c, nil))

	require.Equal(t, 1, pool.CountIdleConnsTesting(addr))
}

func TestLimitMaxIdlePer(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)

	resch := make(chan string)
	gotReq := make(chan bool)
	tr, addr, closeListener := newTestTransport(func(socket transport.Socket) {
		gotReq <- true

		select {
		case <-stop:
			return
		case <-resch:
		}
		require.NoError(t, socket.Recv(&transport.Message{}))
	})
	defer closeListener() // nolint

	pool := newLimitPool(Options{
		Transport:       tr,
		MaxIdleConnsPer: 2,
	})
	defer pool.Close()

	donech := make(chan bool)
	doReq := func() {
		defer func() {
			select {
			case <-stop:
				return
			case donech <- t.Failed():
			}
		}()
		conn, err := pool.Get(addr)
		require.NoError(t, err)
		require.NoError(t, conn.Send(&transport.Message{Body: []byte("hello world")}))
		require.NoError(t, pool.Release(conn, nil))
	}

	go doReq() // idleWait +1, connsWait +1, dialConn
	go doReq() // idleWait +1, connsWait +1, dialConn
	go doReq() // idleWait +1, connsWait +1, dialConn

	<-gotReq
	<-gotReq
	<-gotReq

	require.Equal(t, 0, pool.CountIdleConnsTesting(addr))

	resch <- "res1"
	<-donech
	require.Equal(t, 1, pool.CountIdleConnsTesting(addr))

	resch <- "res2"
	<-donech
	require.Equal(t, 2, pool.CountIdleConnsTesting(addr))

	resch <- "res3"
	<-donech
	require.Equal(t, 2, pool.CountIdleConnsTesting(addr))
}

func TestLimitMaxConnsAndTimeout(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)

	resch := make(chan string)
	gotReq := make(chan bool)
	tr, addr, closeListener := newTestTransport(func(socket transport.Socket) {
		gotReq <- true
		select {
		case <-stop:
			return
		case <-resch:
		}
		require.NoError(t, socket.Recv(&transport.Message{}))
	})
	defer closeListener() // nolint
	pool := newLimitPool(Options{
		Transport:   tr,
		MaxConnsPer: 2,
	})
	defer pool.Close()

	timeout := time.Second

	donech := make(chan bool)
	doReq := func() {
		defer func() {
			select {
			case <-stop:
				return
			case donech <- t.Failed():
			}
		}()
		conn, err := pool.Get(
			addr,
			transport.DialOption(func(opts *transport.DialOptions) {
				opts.Timeout = timeout
			}),
		)
		if err != nil {
			return
		}
		require.NoError(t, conn.Send(&transport.Message{Body: []byte("hello world")}))
		require.NoError(t, pool.Release(conn, nil))
	}

	go doReq() // nolint
	go doReq() // nolint
	go doReq()

	<-gotReq
	<-gotReq

	ticker := time.NewTicker(timeout * 2)
	defer ticker.Stop()
	select {
	case <-donech:
	case <-ticker.C:
		require.NoError(t, errors.New("should Pool.Get timeout"))
	}
}

func (p *limitPool) CountIdleConnsTesting(addr string) int {
	p.idleMu.Lock()
	defer p.idleMu.Unlock()
	key := fmt.Sprintf("%s|%s", p.tr.String(), addr)
	return len(p.idleConns[key])
}

func (p *limitPool) CountIdleWaitTesting(addr string) int {
	p.idleMu.Lock()
	defer p.idleMu.Unlock()
	key := fmt.Sprintf("%s|%s", p.tr.String(), addr)
	q, ok := p.idlePerWait[key]
	if !ok {
		return 0
	}
	return q.len()
}

func (p *limitPool) CountConnWaitTesting(addr string) int {
	p.connsMu.Lock()
	defer p.connsMu.Unlock()
	key := fmt.Sprintf("%s|%s", p.tr.String(), addr)
	q, ok := p.connsPerWait[key]
	if !ok {
		return 0
	}
	return q.len()
}

func newTestTransport(handler func(transport.Socket)) (transport.Transport, string, func() error) {
	tr := transport.NewMemoryTransport()
	l, err := tr.Listen(":0")
	if err != nil {
		panic(fmt.Errorf("transport %s listen at :0 %w", tr.String(), err))
	}
	go func() {
		for {
			if err := l.Accept(handler); err != nil {
				return
			}
		}
	}()
	return tr, l.Addr(), l.Close
}

var remoteHandler = func(socket transport.Socket) {
	var msg transport.Message
	if err := socket.Recv(&msg); err != nil {
		panic(fmt.Errorf("transport recv %w", err))
	}

	if err := socket.Send(&transport.Message{
		Body: []byte(socket.Remote()),
	}); err != nil {
		panic(fmt.Errorf("transport send %w", err))
	}
}
