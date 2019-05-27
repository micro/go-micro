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
	"github.com/micro/go-micro/registry"
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

	// TODO: remove legacy codec list
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

func getHeaders(m *codec.Message) {
	get := func(hdr string) string {
		if hd := m.Header[hdr]; len(hd) > 0 {
			return hd
		}
		// old
		return m.Header["X-"+hdr]
	}

	// check error in header
	if len(m.Error) == 0 {
		m.Error = get("Micro-Error")
	}

	// check endpoint in header
	if len(m.Endpoint) == 0 {
		m.Endpoint = get("Micro-Endpoint")
	}

	// check method in header
	if len(m.Method) == 0 {
		m.Method = get("Micro-Method")
	}

	if len(m.Id) == 0 {
		m.Id = get("Micro-Id")
	}
}

func setHeaders(m *codec.Message) {
	set := func(hdr, v string) {
		if len(v) == 0 {
			return
		}
		m.Header[hdr] = v
		m.Header["X-"+hdr] = v
	}

	set("Micro-Id", m.Id)
	set("Micro-Service", m.Target)
	set("Micro-Method", m.Method)
	set("Micro-Endpoint", m.Endpoint)
}

// setupProtocol sets up the old protocol
func setupProtocol(msg *transport.Message, node *registry.Node) codec.NewCodec {
	protocol := node.Metadata["protocol"]

	// got protocol
	if len(protocol) > 0 {
		return nil
	}

	// no protocol use old codecs
	switch msg.Header["Content-Type"] {
	case "application/json":
		msg.Header["Content-Type"] = "application/json-rpc"
	case "application/protobuf":
		msg.Header["Content-Type"] = "application/proto-rpc"
	}

	// now return codec
	return defaultCodecs[msg.Header["Content-Type"]]
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
	setHeaders(m)

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

func (c *rpcCodec) ReadHeader(m *codec.Message, r codec.MessageType) error {
	var tm transport.Message

	// read message from transport
	if err := c.client.Recv(&tm); err != nil {
		return errors.InternalServerError("go.micro.client.transport", err.Error())
	}

	c.buf.rbuf.Reset()
	c.buf.rbuf.Write(tm.Body)

	// set headers from transport
	m.Header = tm.Header

	// read header
	err := c.codec.ReadHeader(m, r)

	// get headers
	getHeaders(m)

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
