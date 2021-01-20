// Package utp implements a utp transport
package utp

import (
	"bufio"
	"encoding/gob"
	"net"
	"time"

	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/transport"
)

type utpTransport struct {
	opts transport.Options
}

type utpListener struct {
	t    time.Duration
	l    net.Listener
	opts transport.ListenOptions
}

type utpClient struct {
	dialOpts transport.DialOptions
	conn     net.Conn
	enc      *gob.Encoder
	dec      *gob.Decoder
	encBuf   *bufio.Writer
	timeout  time.Duration
}

type utpSocket struct {
	conn    net.Conn
	enc     *gob.Encoder
	dec     *gob.Decoder
	encBuf  *bufio.Writer
	timeout time.Duration
}

func init() {
	cmd.DefaultTransports["utp"] = NewTransport
}

func NewTransport(opts ...transport.Option) transport.Transport {
	var options transport.Options
	for _, o := range opts {
		o(&options)
	}
	return &utpTransport{opts: options}
}
