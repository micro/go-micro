package tunnel

import (
	"sync"

	"github.com/micro/go-micro/transport"
)

type tun struct {
	options Options
	sync.RWMutex
	connected bool
	closed    chan bool
}

func newTunnel(opts ...Option) Tunnel {
	// initialize default options
	options := DefaultOptions()

	for _, o := range opts {
		o(&options)
	}

	t := &tun{
		options: options,
		closed:  make(chan bool),
	}

	return t
}

func (t *tun) Id() string {
	return t.options.Id
}

func (t *tun) Address() string {
	return t.options.Address
}

func (t *tun) Transport() transport.Transport {
	return t.options.Transport
}

func (t *tun) Options() transport.Options {
	return transport.Options{}
}

func (t *tun) Connect() error {
	return nil
}

func (t *tun) Close() error {
	return nil
}

func (t *tun) Status() Status {
	select {
	case <-t.closed:
		return Closed
	default:
		return Connected
	}
}

func (t *tun) String() string {
	return "micro"
}
