package server

import (
	"bytes"
	"fmt"
	"github.com/myodc/go-micro/transport"
	rpc "github.com/youtube/vitess/go/rpcplus"
	js "github.com/youtube/vitess/go/rpcplus/jsonrpc"
	pb "github.com/youtube/vitess/go/rpcplus/pbrpc"
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

func newRpcPlusCodec(req *transport.Message, socket transport.Socket) rpc.ServerCodec {
        r := &rpcPlusCodec{
                socket: socket,
                req:    req,
		buf: &readWriteCloser{
			rbuf: bytes.NewBuffer(req.Body),
			wbuf: bytes.NewBuffer(nil),
		},
        }

	switch req.Header["Content-Type"] {
	case "application/octet-stream":
		r.codec = pb.NewServerCodec(r.buf)
	case "application/json":
		r.codec = js.NewServerCodec(r.buf)
	}

	return r
}

func (c *rpcPlusCodec) ReadRequestHeader(r *rpc.Request) error {
	if c.codec == nil {
		return fmt.Errorf("unsupported content type %s", c.req.Header["Content-Type"])
	}
	return c.codec.ReadRequestHeader(r)
}

func (c *rpcPlusCodec) ReadRequestBody(r interface{}) error {
	return c.codec.ReadRequestBody(r)
}

func (c *rpcPlusCodec) WriteResponse(r *rpc.Response, body interface{}, last bool) error {
	if c.codec == nil {
		return fmt.Errorf("unsupported request type: %s", c.req.Header["Content-Type"])
	}
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
