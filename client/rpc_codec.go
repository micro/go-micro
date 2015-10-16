package client

import (
	"bytes"
	"fmt"

	"github.com/myodc/go-micro/transport"
	rpc "github.com/youtube/vitess/go/rpcplus"
	js "github.com/youtube/vitess/go/rpcplus/jsonrpc"
	pb "github.com/youtube/vitess/go/rpcplus/pbrpc"
)

type rpcPlusCodec struct {
	client transport.Client
	codec  rpc.ClientCodec

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

func newRpcPlusCodec(req *transport.Message, client transport.Client) *rpcPlusCodec {
	r := &rpcPlusCodec{
		req:    req,
		client: client,
		buf: &readWriteCloser{
			wbuf: bytes.NewBuffer(nil),
			rbuf: bytes.NewBuffer(nil),
		},
	}

	switch req.Header["Content-Type"] {
	case "application/octet-stream":
		r.codec = pb.NewClientCodec(r.buf)
	case "application/json":
		r.codec = js.NewClientCodec(r.buf)
	}

	return r
}

func (c *rpcPlusCodec) WriteRequest(req *rpc.Request, body interface{}) error {
	if c.codec == nil {
		return fmt.Errorf("unsupported request type: %s", c.req.Header["Content-Type"])
	}

	if err := c.codec.WriteRequest(req, body); err != nil {
		return err
	}

	c.req.Body = c.buf.wbuf.Bytes()
	return c.client.Send(c.req)
}

func (c *rpcPlusCodec) ReadResponseHeader(r *rpc.Response) error {
	var m transport.Message

	if err := c.client.Recv(&m); err != nil {
		return err
	}

	if c.codec == nil {
		return fmt.Errorf("%s", string(m.Body))
	}

	c.buf.rbuf.Reset()
	c.buf.rbuf.Write(m.Body)
	return c.codec.ReadResponseHeader(r)
}

func (c *rpcPlusCodec) ReadResponseBody(r interface{}) error {
	return c.codec.ReadResponseBody(r)
}

func (c *rpcPlusCodec) Close() error {
	c.buf.Close()
	return c.client.Close()
}
