package client

import (
	"bytes"
	"fmt"

	"github.com/kynrai/go-micro/transport"
	rpc "github.com/youtube/vitess/go/rpcplus"
	js "github.com/youtube/vitess/go/rpcplus/jsonrpc"
	pb "github.com/youtube/vitess/go/rpcplus/pbrpc"
)

type rpcPlusCodec struct {
	client transport.Client
	codec  rpc.ClientCodec

	req *transport.Message

	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
}

func newRpcPlusCodec(req *transport.Message, client transport.Client) *rpcPlusCodec {
	return &rpcPlusCodec{
		req:    req,
		client: client,
		wbuf:   bytes.NewBuffer(nil),
		rbuf:   bytes.NewBuffer(nil),
	}
}

func (c *rpcPlusCodec) WriteRequest(req *rpc.Request, body interface{}) error {
	c.wbuf.Reset()
	buf := &buffer{c.wbuf}

	var cc rpc.ClientCodec
	switch c.req.Header["Content-Type"] {
	case "application/octet-stream":
		cc = pb.NewClientCodec(buf)
	case "application/json":
		cc = js.NewClientCodec(buf)
	default:
		return fmt.Errorf("unsupported request type: %s", c.req.Header["Content-Type"])
	}

	if err := cc.WriteRequest(req, body); err != nil {
		return err
	}

	c.req.Body = c.wbuf.Bytes()
	return c.client.Send(c.req)
}

func (c *rpcPlusCodec) ReadResponseHeader(r *rpc.Response) error {
	var m transport.Message

	if err := c.client.Recv(&m); err != nil {
		return err
	}

	c.rbuf.Reset()
	c.rbuf.Write(m.Body)
	buf := &buffer{c.rbuf}

	switch m.Header["Content-Type"] {
	case "application/octet-stream":
		c.codec = pb.NewClientCodec(buf)
	case "application/json":
		c.codec = js.NewClientCodec(buf)
	default:
		return fmt.Errorf("%s", string(m.Body))
	}

	return c.codec.ReadResponseHeader(r)
}

func (c *rpcPlusCodec) ReadResponseBody(r interface{}) error {
	return c.codec.ReadResponseBody(r)
}

func (c *rpcPlusCodec) Close() error {
	c.rbuf.Reset()
	c.wbuf.Reset()
	return c.client.Close()
}
