// Package kcp is a KCP protocol implementation of the transport
package kcp

import (
	"encoding/gob"
	"net"

	"github.com/micro/go-micro/transport"
	"github.com/xtaci/kcp-go"
)

type kcpSocket struct {
	conn net.Conn
	enc  *gob.Encoder
	dec  *gob.Decoder
}

type kcpTransport struct {
	opts transport.Options
}

type kcpClient struct {
	*kcpSocket
	t    *kcpTransport
	opts transport.DialOptions
}

type kcpListener struct {
	l    net.Listener
	t    *kcpTransport
	opts transport.ListenOptions
}

func (k *kcpClient) Close() error {
	return k.kcpSocket.conn.Close()
}

func (k *kcpSocket) Recv(m *transport.Message) error {
	return k.dec.Decode(&m)
}

func (k *kcpSocket) Send(m *transport.Message) error {
	return k.enc.Encode(m)
}

func (k *kcpSocket) Close() error {
	return k.conn.Close()
}

func (k *kcpSocket) Local() string {
	return k.conn.LocalAddr().String()
}

func (k *kcpSocket) Remote() string {
	return k.conn.RemoteAddr().String()
}

func (k *kcpListener) Addr() string {
	return k.l.Addr().String()
}

func (k *kcpListener) Close() error {
	return k.l.Close()
}

func (k *kcpListener) Accept(fn func(transport.Socket)) error {
	for {
		conn, err := k.l.Accept()
		if err != nil {
			return err
		}

		go func() {
			fn(&kcpSocket{
				conn: conn,
				enc:  gob.NewEncoder(conn),
				dec:  gob.NewDecoder(conn),
			})
		}()
	}
}

func (k *kcpTransport) Init(opts ...transport.Option) error {
	for _, o := range opts {
		o(&k.opts)
	}
	return nil
}

func (k *kcpTransport) Options() transport.Options {
	return k.opts
}

func (k *kcpTransport) Dial(addr string, opts ...transport.DialOption) (transport.Client, error) {
	var options transport.DialOptions
	for _, o := range opts {
		o(&options)
	}

	conn, err := kcp.Dial(addr)
	if err != nil {
		return nil, err
	}

	enc := gob.NewEncoder(conn)
	dec := gob.NewDecoder(conn)

	return &kcpClient{
		&kcpSocket{
			conn: conn,
			enc:  enc,
			dec:  dec,
		},
		k,
		options,
	}, nil
}

func (k *kcpTransport) Listen(addr string, opts ...transport.ListenOption) (transport.Listener, error) {
	var options transport.ListenOptions
	for _, o := range opts {
		o(&options)
	}

	l, err := kcp.Listen(addr)
	if err != nil {
		return nil, err
	}

	return &kcpListener{
		l:    l,
		t:    k,
		opts: options,
	}, nil
}

func (k *kcpTransport) String() string {
	return "kcp"
}

func NewTransport(opts ...transport.Option) transport.Transport {
	options := transport.Options{}

	for _, o := range opts {
		o(&options)
	}

	return &kcpTransport{
		opts: options,
	}
}
