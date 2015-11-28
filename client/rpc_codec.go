package client

import (
	"bytes"

	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/codec/jsonrpc"
	"github.com/micro/go-micro/codec/protorpc"
	"github.com/micro/go-micro/transport"
	rpc "github.com/youtube/vitess/go/rpcplus"
)

type rpcPlusCodec struct {
	client transport.Client
	codec  codec.Codec

	req *transport.Message
	buf *readWriteCloser
}

type readWriteCloser struct {
	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
}

var (
	defaultContentType = "application/octet-stream"

	defaultCodecs = map[string]codec.NewCodec{
		"application/json":         jsonrpc.NewCodec,
		"application/json-rpc":     jsonrpc.NewCodec,
		"application/protobuf":     protorpc.NewCodec,
		"application/proto-rpc":    protorpc.NewCodec,
		"application/octet-stream": protorpc.NewCodec,
	}
)

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

func newRpcPlusCodec(req *transport.Message, client transport.Client, c codec.NewCodec) *rpcPlusCodec {
	rwc := &readWriteCloser{
		wbuf: bytes.NewBuffer(nil),
		rbuf: bytes.NewBuffer(nil),
	}
	r := &rpcPlusCodec{
		buf:    rwc,
		client: client,
		codec:  c(rwc),
		req:    req,
	}
	return r
}

func (c *rpcPlusCodec) WriteRequest(req *rpc.Request, body interface{}) error {
	m := &codec.Message{
		Id:     req.Seq,
		Method: req.ServiceMethod,
		Type:   codec.Request,
	}
	if err := c.codec.Write(m, body); err != nil {
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
	c.buf.rbuf.Reset()
	c.buf.rbuf.Write(m.Body)
	var me codec.Message
	err := c.codec.ReadHeader(&me, codec.Response)
	r.ServiceMethod = me.Method
	r.Seq = me.Id
	r.Error = me.Error
	return err
}

func (c *rpcPlusCodec) ReadResponseBody(b interface{}) error {
	return c.codec.ReadBody(b)
}

func (c *rpcPlusCodec) Close() error {
	c.buf.Close()
	c.codec.Close()
	return c.client.Close()
}
