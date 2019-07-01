package mucp

import (
	"github.com/micro/go-micro/transport"
)

type listener struct {
	// stream id
	id string
	// address of the listener
	addr string
	// close channel
	closed chan bool
	// accept socket
	accept chan *socket
}

func (n *listener) Addr() string {
	return n.addr
}

func (n *listener) Close() error {
	select {
	case <-n.closed:
	default:
		close(n.closed)
	}
	return nil
}

func (n *listener) Accept(fn func(s transport.Socket)) error {
	for {
		select {
		case <-n.closed:
			return nil
		case s, ok := <-n.accept:
			if !ok {
				return nil
			}
			go fn(s)
		}
	}
	return nil
}
