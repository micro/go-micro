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
	// stops the link
	closed chan bool
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

	// keep an error count on the link
	errCount int
}

func newLink(s transport.Socket) *link {
	l := &link{
		Socket:        s,
		id:            uuid.New().String(),
		channels:      make(map[string]time.Time),
		closed:        make(chan bool),
		lastKeepAlive: time.Now(),
	}
	go l.expiry()
	return l
}

// watches the channel expiry
func (l *link) expiry() {
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

func (l *link) Send(m *transport.Message) error {
	err := l.Socket.Send(m)

	l.Lock()
	defer l.Unlock()

	// if theres no error reset the counter
	if err == nil {
		l.errCount = 0
	}

	// otherwise increment the counter
	l.errCount++

	return err
}

func (l *link) Status() string {
	select {
	case <-l.closed:
		return "closed"
	default:
		l.RLock()
		defer l.RUnlock()
		if l.errCount > 3 {
			return "error"
		}
		return "connected"
	}
}
