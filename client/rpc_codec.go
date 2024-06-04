package client

import (
	"bytes"
	errs "errors"

	"go-micro.dev/v5/codec"
	raw "go-micro.dev/v5/codec/bytes"
	"go-micro.dev/v5/codec/grpc"
	"go-micro.dev/v5/codec/json"
	"go-micro.dev/v5/codec/jsonrpc"
	"go-micro.dev/v5/codec/proto"
	"go-micro.dev/v5/codec/protorpc"
	"go-micro.dev/v5/errors"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/transport"
	"go-micro.dev/v5/transport/headers"
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

// errShutdown holds the specific error for closing/closed connections.
var (
	errShutdown = errs.New("connection is shut down")
)

type rpcCodec struct {
	client transport.Client
	codec  codec.Codec

	req *transport.Message
	buf *readWriteCloser

	// signify if its a stream
	stream string
}

type readWriteCloser struct {
	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
}

var (
	// DefaultContentType header.
	DefaultContentType = "application/json"

	// DefaultCodecs map.
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

	// TODO: remove legacy codec list.
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
	set := func(v, hdr string) string {
		if len(v) > 0 {
			return v
		}

		return m.Header[hdr]
	}

	// check error in header
	m.Error = set(m.Error, headers.Error)

	// check endpoint in header
	m.Endpoint = set(m.Endpoint, headers.Endpoint)

	// check method in header
	m.Method = set(m.Method, headers.Method)

	// set the request id
	m.Id = set(m.Id, headers.ID)
}

func setHeaders(m *codec.Message, stream string) {
	set := func(hdr, v string) {
		if len(v) == 0 {
			return
		}

		m.Header[hdr] = v
	}

	set(headers.ID, m.Id)
	set(headers.Request, m.Target)
	set(headers.Method, m.Method)
	set(headers.Endpoint, m.Endpoint)
	set(headers.Error, m.Error)

	if len(stream) > 0 {
		set(headers.Stream, stream)
	}
}

// setupProtocol sets up the old protocol.
func setupProtocol(msg *transport.Message, node *registry.Node) codec.NewCodec {
	protocol := node.Metadata["protocol"]

	// got protocol
	if len(protocol) > 0 {
		return nil
	}

	// processing topic publishing
	if len(msg.Header[headers.Message]) > 0 {
		return nil
	}

	// no protocol use old codecs
	switch msg.Header["Content-Type"] {
	case "application/json":
		msg.Header["Content-Type"] = "application/json-rpc"
	case "application/protobuf":
		msg.Header["Content-Type"] = "application/proto-rpc"
	}

	return defaultCodecs[msg.Header["Content-Type"]]
}

func newRPCCodec(req *transport.Message, client transport.Client, c codec.NewCodec, stream string) codec.Codec {
	rwc := &readWriteCloser{
		wbuf: bytes.NewBuffer(nil),
		rbuf: bytes.NewBuffer(nil),
	}

	return &rpcCodec{
		buf:    rwc,
		client: client,
		codec:  c(rwc),
		req:    req,
		stream: stream,
	}
}

func (c *rpcCodec) Write(message *codec.Message, body interface{}) error {
	c.buf.wbuf.Reset()

	// create header
	if message.Header == nil {
		message.Header = map[string]string{}
	}

	// copy original header
	for k, v := range c.req.Header {
		message.Header[k] = v
	}

	// set the mucp headers
	setHeaders(message, c.stream)

	// if body is bytes Frame don't encode
	if body != nil {
		if b, ok := body.(*raw.Frame); ok {
			// set body
			message.Body = b.Data
		} else {
			// write to codec
			if err := c.codec.Write(message, body); err != nil {
				return errors.InternalServerError("go.micro.client.codec", err.Error())
			}
			// set body
			message.Body = c.buf.wbuf.Bytes()
		}
	}

	// create new transport message
	msg := transport.Message{
		Header: message.Header,
		Body:   message.Body,
	}

	// send the request
	if err := c.client.Send(&msg); err != nil {
		return errors.InternalServerError("go.micro.client.transport", err.Error())
	}

	return nil
}

func (c *rpcCodec) ReadHeader(msg *codec.Message, r codec.MessageType) error {
	var tm transport.Message

	// read message from transport
	if err := c.client.Recv(&tm); err != nil {
		return errors.InternalServerError("go.micro.client.transport", err.Error())
	}

	c.buf.rbuf.Reset()
	c.buf.rbuf.Write(tm.Body)

	// set headers from transport
	msg.Header = tm.Header

	// read header
	err := c.codec.ReadHeader(msg, r)

	// get headers
	getHeaders(msg)

	// return header error
	if err != nil {
		return errors.InternalServerError("go.micro.client.codec", err.Error())
	}

	return nil
}

func (c *rpcCodec) ReadBody(b interface{}) error {
	// read body
	// read raw data
	if v, ok := b.(*raw.Frame); ok {
		v.Data = c.buf.rbuf.Bytes()
		return nil
	}

	if err := c.codec.ReadBody(b); err != nil {
		return errors.InternalServerError("go.micro.client.codec", err.Error())
	}

	return nil
}

func (c *rpcCodec) Close() error {
	if err := c.buf.Close(); err != nil {
		return err
	}

	if err := c.codec.Close(); err != nil {
		return err
	}

	if err := c.client.Close(); err != nil {
		return errors.InternalServerError("go.micro.client.transport", err.Error())
	}

	return nil
}

func (c *rpcCodec) String() string {
	return "rpc"
}
