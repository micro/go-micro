package tunnel

import (
	"io"
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

	// the send queue to the socket
	sendQueue chan *transport.Message
	// the recv queue to the socket
	recvQueue chan *transport.Message

	// determines the cost of the link
	// based on queue length and roundtrip
	length int
	weight int
}

func newLink(s transport.Socket) *link {
	l := &link{
		Socket:    s,
		id:        uuid.New().String(),
		channels:  make(map[string]time.Time),
		closed:    make(chan bool),
		sendQueue: make(chan *transport.Message, 128),
		recvQueue: make(chan *transport.Message, 128),
	}
	go l.expiry()
	go l.process()
	return l
}

// process processes messages on the send and receive queues.
func (l *link) process() {
	go func() {
		for {
			m := new(transport.Message)
			if err := l.Socket.Recv(m); err != nil {
				return
			}

			select {
			case l.recvQueue <- m:
			case <-l.closed:
				return
			}
		}
	}()

	// messages sent
	i := 0
	length := 0

	for {
		select {
		case m := <-l.sendQueue:
			t := time.Now()

			// send the message
			if err := l.Socket.Send(m); err != nil {
				return
			}

			// get header size, body size and time taken
			hl := len(m.Header)
			bl := len(m.Body)
			d := time.Since(t)

			// don't calculate on empty messages
			if hl == 0 && bl == 0 {
				continue
			}

			// increment sent
			i++

			// time take to send some bits and bytes
			td := float64(hl+bl) / float64(d.Nanoseconds())
			// increase the scale
			td += 1

			// judge the length
			length = int(td) / (length + int(td))

			// every 10 messages update length
			if (i % 10) == 1 {
				// cost average the length
				// save it
				l.Lock()
				l.length = length
				l.Unlock()
			}
		case <-l.closed:
			return
		}
	}
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

// length/rate of the link
func (l *link) Length() int {
	l.RLock()
	defer l.RUnlock()
	return l.length
}

// weight checks the size of the queues
func (l *link) Weight() int {
	return len(l.sendQueue) + len(l.recvQueue)
}

// Accept accepts a message on the socket
func (l *link) Recv(m *transport.Message) error {
	select {
	case <-l.closed:
		return io.EOF
	case rm := <-l.recvQueue:
		*m = *rm
		return nil
	}
	// never reach
	return nil
}

// Send sends a message on the socket immediately
func (l *link) Send(m *transport.Message) error {
	select {
	case <-l.closed:
		return io.EOF
	case l.sendQueue <- m:
	}
	return nil
}

func (l *link) Status() string {
	select {
	case <-l.closed:
		return "closed"
	default:
		return "connected"
	}
}
