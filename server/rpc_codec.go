package server

import (
	"bytes"
	"sync"

	"github.com/micro/go-micro/v2/codec"
	raw "github.com/micro/go-micro/v2/codec/bytes"
	"github.com/micro/go-micro/v2/codec/grpc"
	"github.com/micro/go-micro/v2/codec/json"
	"github.com/micro/go-micro/v2/codec/jsonrpc"
	"github.com/micro/go-micro/v2/codec/proto"
	"github.com/micro/go-micro/v2/codec/protorpc"
	"github.com/micro/go-micro/v2/transport"
	"github.com/oxtoacart/bpool"
	"github.com/pkg/errors"
)

type rpcCodec struct {
	socket   transport.Socket
	codec    codec.Codec
	protocol string

	req *transport.Message
	buf *readWriteCloser

	// check if we're the first
	sync.RWMutex
	first chan bool
}

type readWriteCloser struct {
	sync.RWMutex
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

	// TODO: remove legacy codec list
	defaultCodecs = map[string]codec.NewCodec{
		"application/json":         jsonrpc.NewCodec,
		"application/json-rpc":     jsonrpc.NewCodec,
		"application/protobuf":     protorpc.NewCodec,
		"application/proto-rpc":    protorpc.NewCodec,
		"application/octet-stream": protorpc.NewCodec,
	}

	// the local buffer pool
	bufferPool = bpool.NewSizedBufferPool(32, 1)
)

func (rwc *readWriteCloser) Read(p []byte) (n int, err error) {
	rwc.RLock()
	defer rwc.RUnlock()
	return rwc.rbuf.Read(p)
}

func (rwc *readWriteCloser) Write(p []byte) (n int, err error) {
	rwc.Lock()
	defer rwc.Unlock()
	return rwc.wbuf.Write(p)
}

func (rwc *readWriteCloser) Close() error {
	return nil
}

func getHeader(hdr string, md map[string]string) string {
	if hd := md[hdr]; len(hd) > 0 {
		return hd
	}
	return md["X-"+hdr]
}

func getHeaders(m *codec.Message) {
	set := func(v, hdr string) string {
		if len(v) > 0 {
			return v
		}
		return m.Header[hdr]
	}

	m.Id = set(m.Id, "Micro-Id")
	m.Error = set(m.Error, "Micro-Error")
	m.Endpoint = set(m.Endpoint, "Micro-Endpoint")
	m.Method = set(m.Method, "Micro-Method")
	m.Target = set(m.Target, "Micro-Service")

	// TODO: remove this cruft
	if len(m.Endpoint) == 0 {
		m.Endpoint = m.Method
	}
}

func setHeaders(m, r *codec.Message) {
	set := func(hdr, v string) {
		if len(v) == 0 {
			return
		}
		m.Header[hdr] = v
		m.Header["X-"+hdr] = v
	}

	// set headers
	set("Micro-Id", r.Id)
	set("Micro-Service", r.Target)
	set("Micro-Method", r.Method)
	set("Micro-Endpoint", r.Endpoint)
	set("Micro-Error", r.Error)
}

// setupProtocol sets up the old protocol
func setupProtocol(msg *transport.Message) codec.NewCodec {
	service := getHeader("Micro-Service", msg.Header)
	method := getHeader("Micro-Method", msg.Header)
	endpoint := getHeader("Micro-Endpoint", msg.Header)
	protocol := getHeader("Micro-Protocol", msg.Header)
	target := getHeader("Micro-Target", msg.Header)
	topic := getHeader("Micro-Topic", msg.Header)

	// if the protocol exists (mucp) do nothing
	if len(protocol) > 0 {
		return nil
	}

	// newer method of processing messages over transport
	if len(topic) > 0 {
		return nil
	}

	// if no service/method/endpoint then it's the old protocol
	if len(service) == 0 && len(method) == 0 && len(endpoint) == 0 {
		return defaultCodecs[msg.Header["Content-Type"]]
	}

	// old target method specified
	if len(target) > 0 {
		return defaultCodecs[msg.Header["Content-Type"]]
	}

	// no method then set to endpoint
	if len(method) == 0 {
		msg.Header["Micro-Method"] = endpoint
	}

	// no endpoint then set to method
	if len(endpoint) == 0 {
		msg.Header["Micro-Endpoint"] = method
	}

	return nil
}

func newRpcCodec(req *transport.Message, socket transport.Socket, c codec.NewCodec) codec.Codec {
	rwc := &readWriteCloser{
		rbuf: bufferPool.Get(),
		wbuf: bufferPool.Get(),
	}

	r := &rpcCodec{
		buf:      rwc,
		codec:    c(rwc),
		req:      req,
		socket:   socket,
		protocol: "mucp",
		first:    make(chan bool),
	}

	// if grpc pre-load the buffer
	// TODO: remove this terrible hack
	switch r.codec.String() {
	case "grpc":
		// write the body
		rwc.rbuf.Write(req.Body)
		// set the protocol
		r.protocol = "grpc"
	default:
		// first is not preloaded
		close(r.first)
	}

	return r
}

func (c *rpcCodec) ReadHeader(r *codec.Message, t codec.MessageType) error {
	// the initial message
	m := codec.Message{
		Header: c.req.Header,
		Body:   c.req.Body,
	}

	// first message could be pre-loaded
	select {
	case <-c.first:
		// not the first
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

		// set req
		c.req = &tm
	default:
		// we need to lock here to prevent race conditions
		// and we make use of a channel otherwise because
		// this does not result in a context switch
		// locking to check c.first on every call to ReadHeader
		// would otherwise drastically slow the code execution
		c.Lock()
		// recheck before closing because the select statement
		// above is not thread safe, so thread safety here is
		// mandatory
		select {
		case <-c.first:
		default:
			// disable first
			close(c.first)
		}
		// now unlock and we never need this again
		c.Unlock()
	}

	// set some internal things
	getHeaders(&m)

	// read header via codec
	if err := c.codec.ReadHeader(&m, codec.Request); err != nil {
		return err
	}

	// fallback for 0.14 and older
	if len(m.Endpoint) == 0 {
		m.Endpoint = m.Method
	}

	// set message
	*r = m

	return nil
}

func (c *rpcCodec) ReadBody(b interface{}) error {
	// don't read empty body
	if len(c.req.Body) == 0 {
		return nil
	}
	// read raw data
	if v, ok := b.(*raw.Frame); ok {
		v.Data = c.req.Body
		return nil
	}
	// decode the usual way
	return c.codec.ReadBody(b)
}

func (c *rpcCodec) Write(r *codec.Message, b interface{}) error {
	c.buf.wbuf.Reset()

	// create a new message
	m := &codec.Message{
		Target:   r.Target,
		Method:   r.Method,
		Endpoint: r.Endpoint,
		Id:       r.Id,
		Error:    r.Error,
		Type:     r.Type,
		Header:   r.Header,
	}

	if m.Header == nil {
		m.Header = map[string]string{}
	}

	setHeaders(m, r)

	// the body being sent
	var body []byte

	// is it a raw frame?
	if v, ok := b.(*raw.Frame); ok {
		body = v.Data
		// if we have encoded data just send it
	} else if len(r.Body) > 0 {
		body = r.Body
		// write the body to codec
	} else if err := c.codec.Write(m, b); err != nil {
		c.buf.wbuf.Reset()

		// write an error if it failed
		m.Error = errors.Wrapf(err, "Unable to encode body").Error()
		m.Header["Micro-Error"] = m.Error
		// no body to write
		if err := c.codec.Write(m, nil); err != nil {
			return err
		}
	} else {
		// set the body
		body = c.buf.wbuf.Bytes()
	}

	// Set content type if theres content
	if len(body) > 0 {
		m.Header["Content-Type"] = c.req.Header["Content-Type"]
	}

	// send on the socket
	return c.socket.Send(&transport.Message{
		Header: m.Header,
		Body:   body,
	})
}

func (c *rpcCodec) Close() error {
	// close the codec
	c.codec.Close()
	// close the socket
	err := c.socket.Close()
	// put back the buffers
	bufferPool.Put(c.buf.rbuf)
	bufferPool.Put(c.buf.wbuf)
	// return the error
	return err
}

func (c *rpcCodec) String() string {
	return c.protocol
}
