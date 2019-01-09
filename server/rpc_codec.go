package server

import (
	"bytes"

	"github.com/micro/go-micro/codec"
	raw "github.com/micro/go-micro/codec/bytes"
	"github.com/micro/go-micro/codec/grpc"
	"github.com/micro/go-micro/codec/json"
	"github.com/micro/go-micro/codec/jsonrpc"
	"github.com/micro/go-micro/codec/proto"
	"github.com/micro/go-micro/codec/protorpc"
	"github.com/micro/go-micro/transport"
	"github.com/pkg/errors"
)

type rpcCodec struct {
	socket transport.Socket
	codec  codec.Codec
	first  bool

	req *transport.Message
	buf *readWriteCloser
}

type readWriteCloser struct {
	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
}

var (
	DefaultContentType = "application/protobuf"

	DefaultCodecs = map[string]codec.NewCodec{
		"application/grpc":         grpc.NewCodec,
		"application/grpc+json":    grpc.NewCodec,
		"application/grpc+proto":   grpc.NewCodec,
		"application/json":         json.NewCodec,
		"application/json-rpc":     jsonrpc.NewCodec,
		"application/protobuf":     proto.NewCodec,
		"application/proto-rpc":    protorpc.NewCodec,
		"application/octet-stream": raw.NewCodec,
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

func newRpcCodec(req *transport.Message, socket transport.Socket, c codec.NewCodec) codec.Codec {
	rwc := &readWriteCloser{
		rbuf: bytes.NewBuffer(req.Body),
		wbuf: bytes.NewBuffer(nil),
	}
	r := &rpcCodec{
		first:  true,
		buf:    rwc,
		codec:  c(rwc),
		req:    req,
		socket: socket,
	}
	return r
}

func (c *rpcCodec) ReadHeader(r *codec.Message, t codec.MessageType) error {
	// the initieal message
	m := codec.Message{
		Header: c.req.Header,
		Body:   c.req.Body,
	}

	// if its a follow on request read it
	if !c.first {
		var tm transport.Message

		// read off the socket
		if err := c.socket.Recv(&tm); err != nil {
			return err
		}
		// reset the read buffer
		c.buf.rbuf.Reset()

		// write the body to the buffer
		if _, err := c.buf.rbuf.Write(tm.Body); err != nil {
			return err
		}

		// set the message header
		m.Header = tm.Header
		// set the message body
		m.Body = tm.Body
	}

	// no longer first read
	c.first = false

	// set some internal things
	m.Target = m.Header["X-Micro-Service"]
	m.Method = m.Header["X-Micro-Method"]
	m.Id = m.Header["X-Micro-Id"]

	// read header via codec
	err := c.codec.ReadHeader(&m, codec.Request)

	// set the method/id
	r.Method = m.Method
	r.Id = m.Id

	return err
}

func (c *rpcCodec) ReadBody(b interface{}) error {
	return c.codec.ReadBody(b)
}

func (c *rpcCodec) Write(r *codec.Message, b interface{}) error {
	c.buf.wbuf.Reset()

	// create a new message
	m := &codec.Message{
		Method: r.Method,
		Id:     r.Id,
		Error:  r.Error,
		Type:   r.Type,
		Header: map[string]string{
			"X-Micro-Id":     r.Id,
			"X-Micro-Method": r.Method,
			"X-Micro-Error":  r.Error,
			"Content-Type":   c.req.Header["Content-Type"],
		},
	}

	// the body being sent
	var body []byte

	// if we have encoded data just send it
	if len(r.Body) > 0 {
		body = r.Body
		// write to the body
	} else if err := c.codec.Write(m, b); err != nil {
		c.buf.wbuf.Reset()

		// write an error if it failed
		m.Error = errors.Wrapf(err, "Unable to encode body").Error()
		m.Header["X-Micro-Error"] = m.Error
		// no body to write
		if err := c.codec.Write(m, nil); err != nil {
			return err
		}
		// write the body
	} else {
		// set the body
		body = c.buf.wbuf.Bytes()
	}

	// send on the socket
	return c.socket.Send(&transport.Message{
		Header: m.Header,
		Body:   body,
	})
}

func (c *rpcCodec) Close() error {
	c.buf.Close()
	c.codec.Close()
	return c.socket.Close()
}

func (c *rpcCodec) String() string {
	return "rpc"
}
