package tunnel

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/transport"
)

type link struct {
	transport.Socket

	sync.RWMutex

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
	// channels keeps a mapping of channels and last seen
	channels map[string]time.Time
	// stop the link
	closed chan bool
}

func newLink(s transport.Socket) *link {
	l := &link{
		Socket:   s,
		id:       uuid.New().String(),
		channels: make(map[string]time.Time),
		closed:   make(chan bool),
	}
	go l.run()
	return l
}

func (l *link) run() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()

	for {
		select {
		case <-l.closed:
			return
		case <-t.C:
			// drop any channel mappings older than 2 minutes
			var kill []string
			killTime := time.Minute * 2

			l.RLock()
			for ch, t := range l.channels {
				if d := time.Since(t); d > killTime {
					kill = append(kill, ch)
				}
			}
			l.RUnlock()

			// if nothing to kill don't bother with a wasted lock
			if len(kill) == 0 {
				continue
			}

			// kill the channels!
			l.Lock()
			for _, ch := range kill {
				delete(l.channels, ch)
			}
			l.Unlock()
		}
	}
}

func (l *link) Id() string {
	l.RLock()
	defer l.RUnlock()

	return l.id
}

func (l *link) Close() error {
	select {
	case <-l.closed:
		return nil
	default:
		close(l.closed)
		return nil
	}

	return nil
}
