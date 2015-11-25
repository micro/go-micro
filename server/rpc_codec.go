package server

import (
	"bytes"

	"github.com/micro/go-micro/transport"
	rpc "github.com/youtube/vitess/go/rpcplus"
)

type rpcPlusCodec struct {
	socket transport.Socket
	codec  rpc.ServerCodec

	req *transport.Message
	buf *readWriteCloser
}

type readWriteCloser struct {
	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
}

func (rwc *readWriteCloser) Read(p []byte) (n int, err error) {
	return rwc.rbuf.Read(p)
}

func (rwc *readWriteCloser) Write(p []byte) (n int, err error) {
	return rwc.wbuf.Write(p)
}

func (rwc *readWriteCloser) Close() error {
	rwc.rbuf.Reset()
	rwc.wbuf.Reset()
	return nil
}

func newRpcPlusCodec(req *transport.Message, socket transport.Socket, cf codecFunc) rpc.ServerCodec {
	rwc := &readWriteCloser{
		rbuf: bytes.NewBuffer(req.Body),
		wbuf: bytes.NewBuffer(nil),
	}
	r := &rpcPlusCodec{
		buf:    rwc,
		codec:  cf(rwc),
		req:    req,
		socket: socket,
	}
	return r
}

func (c *rpcPlusCodec) ReadRequestHeader(r *rpc.Request) error {
	return c.codec.ReadRequestHeader(r)
}

func (c *rpcPlusCodec) ReadRequestBody(r interface{}) error {
	return c.codec.ReadRequestBody(r)
}

func (c *rpcPlusCodec) WriteResponse(r *rpc.Response, body interface{}, last bool) error {
	c.buf.wbuf.Reset()
	if err := c.codec.WriteResponse(r, body, last); err != nil {
		return err
	}
	return c.socket.Send(&transport.Message{
		Header: map[string]string{"Content-Type": c.req.Header["Content-Type"]},
		Body:   c.buf.wbuf.Bytes(),
	})
}

func (c *rpcPlusCodec) Close() error {
	c.buf.Close()
	return c.socket.Close()
}
