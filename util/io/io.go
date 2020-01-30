// Package io is for io management
package io

import (
	"io"

	"github.com/micro/go-micro/v2/transport"
)

type rwc struct {
	socket transport.Socket
}

func (r *rwc) Read(p []byte) (n int, err error) {
	m := new(transport.Message)
	if err := r.socket.Recv(m); err != nil {
		return 0, err
	}
	copy(p, m.Body)
	return len(m.Body), nil
}

func (r *rwc) Write(p []byte) (n int, err error) {
	err = r.socket.Send(&transport.Message{
		Body: p,
	})
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (r *rwc) Close() error {
	return r.socket.Close()
}

// NewRWC returns a new ReadWriteCloser
func NewRWC(sock transport.Socket) io.ReadWriteCloser {
	return &rwc{sock}
}
