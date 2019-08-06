package tunnel

import (
	"sync"

	"github.com/micro/go-micro/transport"
)

type tun struct {
	sync.RWMutex
	tr        transport.Transport
	options   Options
	connected bool
	closed    chan bool
}

func newTunnel(opts ...Option) Tunnel {
	// initialize default options
	options := DefaultOptions()

	for _, o := range opts {
		o(&options)
	}

	// tunnel transport
	tr := newTransport()

	t := &tun{
		tr:      tr,
		options: options,
		closed:  make(chan bool),
	}

	return t
}

// Id returns tunnel id
func (t *tun) Id() string {
	return t.options.Id
}

// Options returns tunnel options
func (t *tun) Options() Options {
	return t.options
}

// Address returns tunnel listen address
func (t *tun) Address() string {
	return t.options.Address
}

// Transport returns tunnel client transport
func (t *tun) Transport() transport.Transport {
	return t.tr
}

// Connect connects establishes point to point tunnel
func (t *tun) Connect() error {
	return nil
}

// Close closes the tunnel
func (t *tun) Close() error {
	return nil
}

// Status returns tunnel status
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
