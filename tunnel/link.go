package tunnel

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/transport"
)

type link struct {
	sync.RWMutex

	transport.Socket
	// unique id of this link e.g uuid
	// which we define for ourselves
	id string
	// whether its a loopback connection
	// this flag is used by the transport listener
	// which accepts inbound quic connections
	loopback bool
	// whether its actually connected
	// dialled side sets it to connected
	// after sending the message. the
	// listener waits for the connect
	connected bool
	// the last time we received a keepalive
	// on this link from the remote side
	lastKeepAlive time.Time
}

func newLink(s transport.Socket) *link {
	return &link{
		Socket: s,
		id:     uuid.New().String(),
	}
}
