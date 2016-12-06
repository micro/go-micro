package client

import (
	"bytes"
	errs "errors"

	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/codec/jsonrpc"
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

type clientCodec interface {
	WriteRequest(*request, interface{}) error
	ReadResponseHeader(*response) error
	ReadResponseBody(interface{}) error

	Close() error
}

type request struct {
	Service       string
	ServiceMethod string   // format: "Service.Method"
	Seq           uint64   // sequence number chosen by client
	next          *request // for free list in Server
}

type response struct {
	ServiceMethod string    // echoes that of the Request
	Seq           uint64    // echoes that of the request
	Error         string    // error, if any.
	next          *response // for free list in Server
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

func (c *rpcPlusCodec) WriteRequest(req *request, body interface{}) error {
	c.buf.wbuf.Reset()
	m := &codec.Message{
		Id:     req.Seq,
		Target: req.Service,
		Method: req.ServiceMethod,
		Type:   codec.Request,
		Header: map[string]string{},
	}
	if err := c.codec.Write(m, body); err != nil {
		return errors.InternalServerError("go.micro.client.codec", err.Error())
	}
	c.req.Body = c.buf.wbuf.Bytes()
	for k, v := range m.Header {
		c.req.Header[k] = v
	}
	if err := c.client.Send(c.req); err != nil {
		return errors.InternalServerError("go.micro.client.transport", err.Error())
	}
	return nil
}

func (c *rpcPlusCodec) ReadResponseHeader(r *response) error {
	var m transport.Message
	if err := c.client.Recv(&m); err != nil {
		return errors.InternalServerError("go.micro.client.transport", err.Error())
	}
	c.buf.rbuf.Reset()
	c.buf.rbuf.Write(m.Body)
	var me codec.Message
	err := c.codec.ReadHeader(&me, codec.Response)
	r.ServiceMethod = me.Method
	r.Seq = me.Id
	r.Error = me.Error
	if err != nil {
		return errors.InternalServerError("go.micro.client.codec", err.Error())
	}
	return nil
}

func (c *rpcPlusCodec) ReadResponseBody(b interface{}) error {
	if err := c.codec.ReadBody(b); err != nil {
		return errors.InternalServerError("go.micro.client.codec", err.Error())
	}
	return nil
}

func (c *rpcPlusCodec) Close() error {
	c.buf.Close()
	c.codec.Close()
	if err := c.client.Close(); err != nil {
		return errors.InternalServerError("go.micro.client.transport", err.Error())
	}
	return nil
}
