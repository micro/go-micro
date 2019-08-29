package tunnel

import (
	"io"

	"github.com/micro/go-micro/util/log"
)

type tunListener struct {
	// address of the listener
	addr string
	// the accept channel
	accept chan *socket
	// the channel to close
	closed chan bool
	// the tunnel closed channel
	tunClosed chan bool
	// the connection
	conn Conn
	// the listener socket
	socket *socket
}

func (t *tunListener) process() {
	// our connection map for session
	conns := make(map[string]*socket)

	for {
		select {
		case <-t.closed:
			return
		// receive a new message
		case m := <-t.socket.recv:
			// get a socket
			sock, ok := conns[m.session]
			log.Debugf("Tunnel listener received id %s session %s exists: %t", m.id, m.session, ok)
			if !ok {
				// create a new socket session
				sock = &socket{
					// our tunnel id
					id: m.id,
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
					send: t.socket.send,
					// wait
					wait: make(chan bool),
				}

				// save the socket
				conns[m.session] = sock

				// send to accept chan
				select {
				case <-t.closed:
					return
				case t.accept <- sock:
				}
			}

			// send this to the accept chan
			select {
			case <-sock.closed:
				delete(conns, m.session)
			case sock.recv <- m:
				log.Debugf("Tunnel listener sent to recv chan id %s session %s", m.id, m.session)
			}
		}
	}
}

func (t *tunListener) Addr() string {
	return t.addr
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
func (t *tunListener) Accept() (Conn, error) {
	select {
	// if the socket is closed return
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
