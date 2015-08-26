package server

import (
	"bytes"
	"fmt"

	"github.com/kynrai/go-micro/transport"
	rpc "github.com/youtube/vitess/go/rpcplus"
	js "github.com/youtube/vitess/go/rpcplus/jsonrpc"
	pb "github.com/youtube/vitess/go/rpcplus/pbrpc"
)

type rpcPlusCodec struct {
	socket transport.Socket
	codec  rpc.ServerCodec

	req *transport.Message

	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
}

func newRpcPlusCodec(req *transport.Message, socket transport.Socket) *rpcPlusCodec {
	return &rpcPlusCodec{
		socket: socket,
		req:    req,
		wbuf:   bytes.NewBuffer(nil),
		rbuf:   bytes.NewBuffer(nil),
	}
}

func (c *rpcPlusCodec) ReadRequestHeader(r *rpc.Request) error {
	c.rbuf.Reset()
	c.rbuf.Write(c.req.Body)
	buf := &buffer{c.rbuf}

	switch c.req.Header["Content-Type"] {
	case "application/octet-stream":
		c.codec = pb.NewServerCodec(buf)
	case "application/json":
		c.codec = js.NewServerCodec(buf)
	default:
		return fmt.Errorf("unsupported content type %s", c.req.Header["Content-Type"])
	}

	return c.codec.ReadRequestHeader(r)
}

func (c *rpcPlusCodec) ReadRequestBody(r interface{}) error {
	return c.codec.ReadRequestBody(r)
}

func (c *rpcPlusCodec) WriteResponse(r *rpc.Response, body interface{}, last bool) error {
	c.wbuf.Reset()
	buf := &buffer{c.wbuf}

	var cc rpc.ServerCodec
	switch c.req.Header["Content-Type"] {
	case "application/octet-stream":
		cc = pb.NewServerCodec(buf)
	case "application/json":
		cc = js.NewServerCodec(buf)
	default:
		return fmt.Errorf("unsupported request type: %s", c.req.Header["Content-Type"])
	}

	if err := cc.WriteResponse(r, body, last); err != nil {
		return err
	}

	return c.socket.Send(&transport.Message{
		Header: map[string]string{"Content-Type": c.req.Header["Content-Type"]},
		Body:   c.wbuf.Bytes(),
	})

}

func (c *rpcPlusCodec) Close() error {
	c.wbuf.Reset()
	c.rbuf.Reset()
	return c.socket.Close()
}
