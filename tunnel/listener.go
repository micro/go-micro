package tunnel

import (
	"io"
	"time"

	"github.com/micro/go-micro/util/log"
)

type tunListener struct {
	// address of the listener
	channel string
	// the accept channel
	accept chan *session
	// the channel to close
	closed chan bool
	// the tunnel closed channel
	tunClosed chan bool
	// the listener session
	session *session
	// del func to kill listener
	delFunc func()
}

// periodically announce self
func (t *tunListener) announce() {
	tick := time.NewTicker(time.Second * 30)
	defer tick.Stop()

	// first announcement
	t.session.Announce()

	for {
		select {
		case <-tick.C:
			t.session.Announce()
		case <-t.closed:
			return
		}
	}
}

func (t *tunListener) process() {
	// our connection map for session
	conns := make(map[string]*session)

	defer func() {
		// close the sessions
		for _, conn := range conns {
			conn.Close()
		}
	}()

	for {
		select {
		case <-t.closed:
			return
		case <-t.tunClosed:
			t.Close()
			return
		// receive a new message
		case m := <-t.session.recv:
			// get a session
			sess, ok := conns[m.session]
			log.Debugf("Tunnel listener received channel %s session %s exists: %t", m.channel, m.session, ok)
			if !ok {
				switch m.typ {
				case "open", "session":
				default:
					continue
				}

				// create a new session session
				sess = &session{
					// the id of the remote side
					tunnel: m.tunnel,
					// the channel
					channel: m.channel,
					// the session id
					session: m.session,
					// is loopback conn
					loopback: m.loopback,
					// the link the message was received on
					link: m.link,
					// set multicast
					multicast: m.multicast,
					// close chan
					closed: make(chan bool),
					// recv called by the acceptor
					recv: make(chan *message, 128),
					// use the internal send buffer
					send: t.session.send,
					// wait
					wait: make(chan bool),
					// error channel
					errChan: make(chan error, 1),
				}

				// save the session
				conns[m.session] = sess

				select {
				case <-t.closed:
					return
				// send to accept chan
				case t.accept <- sess:
				}
			}

			// an existing session was found

			// received a close message
			switch m.typ {
			case "close":
				select {
				case <-sess.closed:
					// no op
					delete(conns, m.session)
				default:
					// close and delete session
					close(sess.closed)
					delete(conns, m.session)
				}

				// continue
				continue
			case "session":
				// operate on this
			default:
				// non operational type
				continue
			}

			// send this to the accept chan
			select {
			case <-sess.closed:
				delete(conns, m.session)
			case sess.recv <- m:
				log.Debugf("Tunnel listener sent to recv chan channel %s session %s", m.channel, m.session)
			}
		}
	}
}

func (t *tunListener) Channel() string {
	return t.channel
}

// Close closes tunnel listener
func (t *tunListener) Close() error {
	select {
	case <-t.closed:
		return nil
	default:
		// close and delete
		t.delFunc()
		t.session.Close()
		close(t.closed)
	}
	return nil
}

// Everytime accept is called we essentially block till we get a new connection
func (t *tunListener) Accept() (Session, error) {
	select {
	// if the session is closed return
	case <-t.closed:
		return nil, io.EOF
	case <-t.tunClosed:
		// close the listener when the tunnel closes
		return nil, io.EOF
	// wait for a new connection
	case c, ok := <-t.accept:
		// check if the accept chan is closed
		if !ok {
			return nil, io.EOF
		}
		// send back the accept
		if err := c.Accept(); err != nil {
			return nil, err
		}
		return c, nil
	}
	return nil, nil
}
