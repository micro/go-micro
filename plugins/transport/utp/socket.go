package utp

import (
	"errors"
	"time"

	"github.com/asim/go-micro/v3/transport"
)

func (u *utpSocket) Local() string {
	return u.conn.LocalAddr().String()
}

func (u *utpSocket) Remote() string {
	return u.conn.RemoteAddr().String()
}

func (u *utpSocket) Recv(m *transport.Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	// set timeout if its greater than 0
	if u.timeout > time.Duration(0) {
		u.conn.SetDeadline(time.Now().Add(u.timeout))
	}

	return u.dec.Decode(&m)
}

func (u *utpSocket) Send(m *transport.Message) error {
	// set timeout if its greater than 0
	if u.timeout > time.Duration(0) {
		u.conn.SetDeadline(time.Now().Add(u.timeout))
	}
	if err := u.enc.Encode(m); err != nil {
		return err
	}
	return u.encBuf.Flush()
}

func (u *utpSocket) Close() error {
	return u.conn.Close()
}
