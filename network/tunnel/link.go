package tunnel

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/transport"
)

type link struct {
	transport.Socket

	// transport to use for connections
	transport transport.Transport

	sync.RWMutex

	// stops the link
	closed chan bool
	// metric used to track metrics
	metric chan *metric
	// link state channel for testing link
	state chan *packet
	// send queue for sending packets
	sendQueue chan *packet
	// receive queue for receiving packets
	recvQueue chan *packet
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
	// the weighted moving average roundtrip
	length int64
	// weighted moving average of bits flowing
	rate float64
	// keep an error count on the link
	errCount int
}

// packet send over link
type packet struct {
	// message to send or received
	message *transport.Message

	// status returned when sent
	status chan error

	// receive related error
	err error
}

// metric is used to record link rate
type metric struct {
	// amount of data sent
	data int
	// time taken to send
	duration time.Duration
	// if an error occurred
	status error
}

var (
	// the 4 byte 0 packet sent to determine the link state
	linkRequest = []byte{0, 0, 0, 0}
	// the 4 byte 1 filled packet sent to determine link state
	linkResponse = []byte{1, 1, 1, 1}

	ErrLinkConnectTimeout = errors.New("link connect timeout")
)

func newLink(s transport.Socket) *link {
	l := &link{
		Socket:        s,
		id:            uuid.New().String(),
		lastKeepAlive: time.Now(),
		closed:        make(chan bool),
		channels:      make(map[string]time.Time),
		state:         make(chan *packet, 64),
		sendQueue:     make(chan *packet, 128),
		recvQueue:     make(chan *packet, 128),
		metric:        make(chan *metric, 128),
	}

	// process inbound/outbound packets
	go l.process()
	// manage the link state
	go l.manage()

	return l
}

// setRate sets the bits per second rate as a float64
func (l *link) setRate(bits int64, delta time.Duration) {
	// rate of send in bits per nanosecond
	rate := float64(bits) / float64(delta.Nanoseconds())

	// default the rate if its zero
	if l.rate == 0 {
		// rate per second
		l.rate = rate * 1e9
	} else {
		// set new rate per second
		l.rate = 0.8*l.rate + 0.2*(rate*1e9)
	}
}

// setRTT sets a nanosecond based moving average roundtrip time for the link
func (l *link) setRTT(d time.Duration) {
	l.Lock()

	if l.length <= 0 {
		l.length = d.Nanoseconds()
		l.Unlock()
		return
	}

	// https://fishi.devtail.io/weblog/2015/04/12/measuring-bandwidth-and-round-trip-time-tcp-connection-inside-application-layer/
	length := 0.8*float64(l.length) + 0.2*float64(d.Nanoseconds())
	// set new length
	l.length = int64(length)

	l.Unlock()
}

func (l *link) delChannel(ch string) {
	l.Lock()
	delete(l.channels, ch)
	l.Unlock()
}

func (l *link) getChannel(ch string) time.Time {
	l.RLock()
	t := l.channels[ch]
	l.RUnlock()
	return t
}

func (l *link) setChannel(channels ...string) {
	l.Lock()
	for _, ch := range channels {
		l.channels[ch] = time.Now()
	}
	l.Unlock()
}

// set the keepalive time
func (l *link) keepalive() {
	l.Lock()
	l.lastKeepAlive = time.Now()
	l.Unlock()
}

// process deals with the send queue
func (l *link) process() {
	// receive messages
	go func() {
		for {
			m := new(transport.Message)
			err := l.recv(m)
			if err != nil {
				// record the metric
				select {
				case l.metric <- &metric{status: err}:
				default:
				}
			}

			// process new received message

			pk := &packet{message: m, err: err}

			// this is our link state packet
			if m.Header["Micro-Method"] == "link" {
				// process link state message
				select {
				case l.state <- pk:
				case <-l.closed:
					return
				default:
				}
				continue
			}

			// process all messages as is

			select {
			case l.recvQueue <- pk:
			case <-l.closed:
				return
			}
		}
	}()

	// send messages

	for {
		select {
		case pk := <-l.sendQueue:
			// send the message
			select {
			case pk.status <- l.send(pk.message):
			case <-l.closed:
				return
			}
		case <-l.closed:
			return
		}
	}
}

// manage manages the link state including rtt packets and channel mapping expiry
func (l *link) manage() {
	// tick over every minute to expire and fire rtt packets
	t1 := time.NewTicker(time.Minute)
	defer t1.Stop()

	// used to batch update link metrics
	t2 := time.NewTicker(time.Second * 5)
	defer t2.Stop()

	// get link id
	linkId := l.Id()

	// used to send link state packets
	send := func(b []byte) error {
		return l.Send(&transport.Message{
			Header: map[string]string{
				"Micro-Method":  "link",
				"Micro-Link-Id": linkId,
			}, Body: b,
		})
	}

	// set time now
	now := time.Now()

	// send the initial rtt request packet
	send(linkRequest)

	for {
		select {
		// exit if closed
		case <-l.closed:
			return
		// process link state rtt packets
		case p := <-l.state:
			if p.err != nil {
				continue
			}
			// check the type of message
			switch {
			case bytes.Equal(p.message.Body, linkRequest):
				if logger.V(logger.TraceLevel, log) {
					log.Tracef("Link %s received link request", linkId)
				}
				// send response
				if err := send(linkResponse); err != nil {
					l.Lock()
					l.errCount++
					l.Unlock()
				}
			case bytes.Equal(p.message.Body, linkResponse):
				// set round trip time
				d := time.Since(now)
				if logger.V(logger.TraceLevel, log) {
					log.Tracef("Link %s received link response in %v", linkId, d)
				}
				// set the RTT
				l.setRTT(d)
			}
		case <-t1.C:
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

			// fire off a link state rtt packet
			now = time.Now()
			send(linkRequest)
		case <-t2.C:
			// get a batch of metrics
			batch := l.batch()

			// skip if there's no metrics
			if len(batch) == 0 {
				continue
			}

			// lock once to record a batch
			l.Lock()
			for _, metric := range batch {
				l.record(metric)
			}
			l.Unlock()
		}
	}
}

func (l *link) batch() []*metric {
	var metrics []*metric

	// pull all the metrics
	for {
		select {
		case m := <-l.metric:
			metrics = append(metrics, m)
		// non blocking return
		default:
			return metrics
		}
	}
}

func (l *link) record(m *metric) {
	// there's an error increment the counter and bail
	if m.status != nil {
		l.errCount++
		return
	}

	// reset the counter
	l.errCount = 0

	// calculate based on data
	if m.data > 0 {
		// bit sent
		bits := m.data * 1024

		// set the rate
		l.setRate(int64(bits), m.duration)
	}
}

func (l *link) send(m *transport.Message) error {
	if m.Header == nil {
		m.Header = make(map[string]string)
	}
	// send the message
	return l.Socket.Send(m)
}

// recv a message on the link
func (l *link) recv(m *transport.Message) error {
	if m.Header == nil {
		m.Header = make(map[string]string)
	}
	// receive the transport message
	return l.Socket.Recv(m)
}

// Delay is the current load on the link
func (l *link) Delay() int64 {
	return int64(len(l.sendQueue) + len(l.recvQueue))
}

// Current transfer rate as bits per second (lower is better)
func (l *link) Rate() float64 {
	l.RLock()
	r := l.rate
	l.RUnlock()
	return r
}

func (l *link) Loopback() bool {
	l.RLock()
	lo := l.loopback
	l.RUnlock()
	return lo
}

// Length returns the roundtrip time as nanoseconds (lower is better).
// Returns 0 where no measurement has been taken.
func (l *link) Length() int64 {
	l.RLock()
	length := l.length
	l.RUnlock()
	return length
}

func (l *link) Id() string {
	l.RLock()
	id := l.id
	l.RUnlock()
	return id
}

func (l *link) Close() error {
	l.Lock()
	defer l.Unlock()

	select {
	case <-l.closed:
		return nil
	default:
		l.Socket.Close()
		close(l.closed)
	}

	return nil
}

// Send sencs a message on the link
func (l *link) Send(m *transport.Message) error {
	// create a new packet to send over the link
	p := &packet{
		message: m,
		status:  make(chan error, 1),
	}

	// calculate the data sent
	dataSent := len(m.Body)

	// set header length
	for k, v := range m.Header {
		dataSent += (len(k) + len(v))
	}

	// get time now
	now := time.Now()

	// queue the message
	select {
	case l.sendQueue <- p:
		// in the send queue
	case <-l.closed:
		return io.EOF
	}

	// error to use
	var err error

	// wait for response
	select {
	case <-l.closed:
		return io.EOF
	case err = <-p.status:
	}

	// create a metric with
	// time taken, size of package, error status
	mt := &metric{
		data:     dataSent,
		duration: time.Since(now),
		status:   err,
	}

	// pass back a metric
	// do not block
	select {
	case l.metric <- mt:
	default:
	}

	return nil
}

// Accept accepts a message on the socket
func (l *link) Recv(m *transport.Message) error {
	select {
	case <-l.closed:
		// check if there's any messages left
		select {
		case pk := <-l.recvQueue:
			// check the packet receive error
			if pk.err != nil {
				return pk.err
			}
			*m = *pk.message
		default:
			return io.EOF
		}
	case pk := <-l.recvQueue:
		// check the packet receive error
		if pk.err != nil {
			return pk.err
		}
		*m = *pk.message
	}
	return nil
}

// State can return connected, closed, error
func (l *link) State() string {
	select {
	case <-l.closed:
		return "closed"
	default:
		l.RLock()
		errCount := l.errCount
		l.RUnlock()

		if errCount > 3 {
			return "error"
		}

		return "connected"
	}
}
