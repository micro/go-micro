package client

import (
	"bytes"
	errs "errors"

	"github.com/micro/go-micro/codec"
	raw "github.com/micro/go-micro/codec/bytes"
	"github.com/micro/go-micro/codec/grpc"
	"github.com/micro/go-micro/codec/json"
	"github.com/micro/go-micro/codec/jsonrpc"
	"github.com/micro/go-micro/codec/proto"
	"github.com/micro/go-micro/codec/protorpc"
	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/transport"
)

const (
	lastStreamResponseError = "EOS"
)

// serverError represents an error that has been returned from
// the remote side of the RPC connection.
type serverError string

func (e serverError) Error() string {
	return string(e)
}

// errShutdown holds the specific error for closing/closed connections
var (
	errShutdown = errs.New("connection is shut down")
)

type rpcCodec struct {
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
	DefaultContentType = "application/protobuf"

	DefaultCodecs = map[string]codec.NewCodec{
		"application/grpc":         grpc.NewCodec,
		"application/grpc+json":    grpc.NewCodec,
		"application/grpc+proto":   grpc.NewCodec,
		"application/protobuf":     proto.NewCodec,
		"application/json":         json.NewCodec,
		"application/json-rpc":     jsonrpc.NewCodec,
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

func newRpcCodec(req *transport.Message, client transport.Client, c codec.NewCodec) codec.Codec {
	rwc := &readWriteCloser{
		wbuf: bytes.NewBuffer(nil),
		rbuf: bytes.NewBuffer(nil),
	}
	r := &rpcCodec{
		buf:    rwc,
		client: client,
		codec:  c(rwc),
		req:    req,
	}
	return r
}

func (c *rpcCodec) Write(m *codec.Message, body interface{}) error {
	c.buf.wbuf.Reset()

	// create header
	if m.Header == nil {
		m.Header = map[string]string{}
	}

	// copy original header
	for k, v := range c.req.Header {
		m.Header[k] = v
	}

	// set the mucp headers
	m.Header["X-Micro-Id"] = m.Id
	m.Header["X-Micro-Service"] = m.Target
	m.Header["X-Micro-Method"] = m.Method
	m.Header["X-Micro-Endpoint"] = m.Endpoint

	// if body is bytes Frame don't encode
	if body != nil {
		b, ok := body.(*raw.Frame)
		if ok {
			// set body
			m.Body = b.Data
			body = nil
		}
	}

	if len(m.Body) == 0 {
		// write to codec
		if err := c.codec.Write(m, body); err != nil {
			return errors.InternalServerError("go.micro.client.codec", err.Error())
		}
		// set body
		m.Body = c.buf.wbuf.Bytes()
	}

	// create new transport message
	msg := transport.Message{
		Header: m.Header,
		Body:   m.Body,
	}

	// send the request
	if err := c.client.Send(&msg); err != nil {
		return errors.InternalServerError("go.micro.client.transport", err.Error())
	}

	return nil
}

func (c *rpcCodec) ReadHeader(wm *codec.Message, r codec.MessageType) error {
	var m transport.Message
	if err := c.client.Recv(&m); err != nil {
		return errors.InternalServerError("go.micro.client.transport", err.Error())
	}
	c.buf.rbuf.Reset()
	c.buf.rbuf.Write(m.Body)

	var me codec.Message
	// set headers
	me.Header = m.Header

	// read header
	err := c.codec.ReadHeader(&me, r)
	wm.Endpoint = me.Endpoint
	wm.Method = me.Method
	wm.Id = me.Id
	wm.Error = me.Error

	// check error in header
	if len(me.Error) == 0 {
		wm.Error = me.Header["X-Micro-Error"]
	}

	// check endpoint in header
	if len(me.Endpoint) == 0 {
		wm.Endpoint = me.Header["X-Micro-Endpoint"]
	}

	// check method in header
	if len(me.Method) == 0 {
		wm.Method = me.Header["X-Micro-Method"]
	}

	if len(me.Id) == 0 {
		wm.Id = me.Header["X-Micro-Id"]
	}

	// return header error
	if err != nil {
		return errors.InternalServerError("go.micro.client.codec", err.Error())
	}

	return nil
}

func (c *rpcCodec) ReadBody(b interface{}) error {
	// read body
	if err := c.codec.ReadBody(b); err != nil {
		return errors.InternalServerError("go.micro.client.codec", err.Error())
	}
	return nil
}

func (c *rpcCodec) Close() error {
	c.buf.Close()
	c.codec.Close()
	if err := c.client.Close(); err != nil {
		return errors.InternalServerError("go.micro.client.transport", err.Error())
	}
	return nil
}

func (c *rpcCodec) String() string {
	return "rpc"
}
