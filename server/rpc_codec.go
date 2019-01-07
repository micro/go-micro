package server

import (
	"bytes"
	"fmt"
	"strconv"

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

	req *transport.Message
	buf *readWriteCloser
}

type readWriteCloser struct {
	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
}

type serverCodec interface {
	ReadHeader(*request, bool) error
	ReadBody(interface{}) error
	Write(*response, interface{}, bool) error
	Close() error
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

func newRpcCodec(req *transport.Message, socket transport.Socket, c codec.NewCodec) serverCodec {
	rwc := &readWriteCloser{
		rbuf: bytes.NewBuffer(req.Body),
		wbuf: bytes.NewBuffer(nil),
	}
	r := &rpcCodec{
		buf:    rwc,
		codec:  c(rwc),
		req:    req,
		socket: socket,
	}
	return r
}

func (c *rpcCodec) ReadHeader(r *request, first bool) error {
	m := codec.Message{Header: c.req.Header}

	if !first {
		var tm transport.Message
		if err := c.socket.Recv(&tm); err != nil {
			return err
		}
		c.buf.rbuf.Reset()
		if _, err := c.buf.rbuf.Write(tm.Body); err != nil {
			return err
		}

		m.Header = tm.Header
	}

	// set some internal things
	m.Target = m.Header["X-Micro-Service"]
	m.Method = m.Header["X-Micro-Method"]

	// set id
	if len(m.Header["X-Micro-Id"]) > 0 {
		id, _ := strconv.ParseInt(m.Header["X-Micro-Id"], 10, 64)
		m.Id = uint64(id)
	}

	// read header via codec
	err := c.codec.ReadHeader(&m, codec.Request)
	r.ServiceMethod = m.Method
	r.Seq = m.Id

	return err
}

func (c *rpcCodec) ReadBody(b interface{}) error {
	return c.codec.ReadBody(b)
}

func (c *rpcCodec) Write(r *response, body interface{}, last bool) error {
	c.buf.wbuf.Reset()
	m := &codec.Message{
		Method: r.ServiceMethod,
		Id:     r.Seq,
		Error:  r.Error,
		Type:   codec.Response,
		Header: map[string]string{
			"X-Micro-Id":     fmt.Sprintf("%d", r.Seq),
			"X-Micro-Method": r.ServiceMethod,
			"X-Micro-Error":  r.Error,
		},
	}
	if err := c.codec.Write(m, body); err != nil {
		c.buf.wbuf.Reset()
		m.Error = errors.Wrapf(err, "Unable to encode body").Error()
		m.Header["X-Micro-Error"] = m.Error
		if err := c.codec.Write(m, nil); err != nil {
			return err
		}
	}

	m.Header["Content-Type"] = c.req.Header["Content-Type"]
	return c.socket.Send(&transport.Message{
		Header: m.Header,
		Body:   c.buf.wbuf.Bytes(),
	})
}

func (c *rpcCodec) Close() error {
	c.buf.Close()
	c.codec.Close()
	return c.socket.Close()
}
