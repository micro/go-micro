package rpc

import (
	"bytes"
	errs "errors"

	"github.com/micro/go-micro/codec"
	raw "github.com/micro/go-micro/codec/bytes"
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
	DefaultCodecs = map[string]codec.NewCodec{
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

func (c *rpcCodec) Write(wm *codec.Message, body interface{}) error {
	c.buf.wbuf.Reset()

	m := &codec.Message{
		Id:       wm.Id,
		Target:   wm.Target,
		Endpoint: wm.Endpoint,
		Type:     codec.Request,
		Header: map[string]string{
			"X-Micro-Id":       wm.Id,
			"X-Micro-Service":  wm.Target,
			"X-Micro-Endpoint": wm.Endpoint,
		},
	}

	if err := c.codec.Write(m, body); err != nil {
		return errors.InternalServerError("go.micro.client.codec", err.Error())
	}

	// set body
	if len(wm.Body) > 0 {
		c.req.Body = wm.Body
	} else {
		c.req.Body = c.buf.wbuf.Bytes()
	}

	// set header
	for k, v := range m.Header {
		c.req.Header[k] = v
	}

	// send the request
	if err := c.client.Send(c.req); err != nil {
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
	wm.Id = me.Id
	wm.Error = me.Error

	// check error in header
	if len(me.Error) == 0 {
		wm.Error = me.Header["X-Micro-Error"]
	}

	// check method in header
	if len(me.Endpoint) == 0 {
		wm.Endpoint = me.Header["X-Micro-Endpoint"]
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
