package tunnel

import (
	"io"

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
}

func (t *tunListener) process() {
	// our connection map for session
	conns := make(map[string]*session)

	for {
		select {
		case <-t.closed:
			return
		// receive a new message
		case m := <-t.session.recv:
			// get a session
			sess, ok := conns[m.session]
			log.Debugf("Tunnel listener received id %s session %s exists: %t", m.id, m.session, ok)
			if !ok {
				// create a new session session
				sess = &session{
					// the id of the remote side
					id: m.id,
					// the channel
					channel: m.channel,
					// the session id
					session: m.session,
					// is loopback conn
					loopback: m.loopback,
					// the link the message was received on
					link: m.link,
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

				// send to accept chan
				select {
				case <-t.closed:
					return
				case t.accept <- sess:
				}
			}

			// send this to the accept chan
			select {
			case <-sess.closed:
				delete(conns, m.session)
			case sess.recv <- m:
				log.Debugf("Tunnel listener sent to recv chan id %s session %s", m.id, m.session)
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
		t.Close()
		return nil, io.EOF
	// wait for a new connection
	case c, ok := <-t.accept:
		if !ok {
			return nil, io.EOF
		}
		return c, nil
	}
	return nil, nil
}
