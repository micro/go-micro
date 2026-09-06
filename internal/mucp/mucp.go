// Package mucp implements the shared wire framing used by the RPC client and
// server. Both sides previously carried their own copy of the mucp protocol;
// this package is the single owner of the framing contract, header
// canonicalization and legacy protocol detection.
package mucp

import (
	"bytes"
	errs "errors"
	"fmt"
	"sync"

	"github.com/oxtoacart/bpool"
	pkgerrors "github.com/pkg/errors"

	"go-micro.dev/v6/codec"
	raw "go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/codec/grpc"
	"go-micro.dev/v6/codec/json"
	"go-micro.dev/v6/codec/jsonrpc"
	"go-micro.dev/v6/codec/proto"
	"go-micro.dev/v6/codec/protorpc"
	merrors "go-micro.dev/v6/errors"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/transport"
	"go-micro.dev/v6/transport/headers"
)

// Conn is the transport surface the framing depends on. Both
// transport.Client and transport.Socket satisfy it structurally.
type Conn interface {
	Send(*transport.Message) error
	Recv(*transport.Message) error
	Close() error
}

// Options is the only knob the framing needs. Everything else is inferred
// from the request message and the codec maps.
type Options struct {
	// Request is the message carrying the request. Its header map is copied
	// into every outbound frame (client) and is the first frame to read
	// (server). NewClient/NewServer may rewrite entries on it for legacy
	// wire conversion. Do not share it across concurrent codecs.
	Request *transport.Message

	// Protocol is the registry node's Metadata["protocol"] value (client
	// only; the selector has already resolved the node). When empty the
	// destination may be a pre-protocol server, so the legacy json-rpc /
	// proto-rpc fallback switches on. A topic publish always skips it.
	Protocol string

	// Stream is the stream id; empty for unary calls.
	Stream string

	// Codecs are preferred codec constructors, consulted before the
	// built-in maps.
	Codecs map[string]codec.NewCodec

	// Domain prefixes the go-micro error envelope for transport/codec
	// failures (client passes "go.micro.client"). Empty returns raw errors.
	Domain string
}

var (
	// DefaultCodecs maps content types to their codecs for the modern (mucp)
	// wire format. Was duplicated identically in client and server.
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

	// legacyCodecs is the 0.14-and-older wire format family.
	legacyCodecs = map[string]codec.NewCodec{
		"application/json":         jsonrpc.NewCodec,
		"application/json-rpc":     jsonrpc.NewCodec,
		"application/protobuf":     protorpc.NewCodec,
		"application/proto-rpc":    protorpc.NewCodec,
		"application/octet-stream": protorpc.NewCodec,
	}

	// bufferPool is the local buffer pool used by the server side.
	bufferPool = bpool.NewSizedBufferPool(32, 1)
)

// Lookup resolves a modern codec constructor by content type, preferring the
// caller-supplied overrides.
func Lookup(contentType string, overrides map[string]codec.NewCodec) (codec.NewCodec, error) {
	if overrides != nil {
		if c, ok := overrides[contentType]; ok {
			return c, nil
		}
	}
	if c, ok := DefaultCodecs[contentType]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("unsupported Content-Type: %s", contentType)
}

// GetHeader reads a header with the canonical X- prefix fallback.
func GetHeader(hdr string, md map[string]string) string {
	if v := md[hdr]; len(v) > 0 {
		return v
	}
	return md["X-"+hdr]
}

// SetupProtocol resolves the legacy-codec fallback for a message. node is the
// registry node on the client side (nil on the server side); the resulting
// codec is nil when the modern wire applies.
func SetupProtocol(msg *transport.Message, node *registry.Node) codec.NewCodec {
	if node == nil {
		return legacyForResponse(Options{Request: msg})
	}
	return legacyForRequest(Options{Request: msg, Protocol: node.Metadata["protocol"]})
}

// NewClient builds a framing codec that writes requests and reads responses
// on conn.
func NewClient(conn Conn, o Options) (codec.Codec, error) {
	if o.Request == nil {
		return nil, errs.New("mucp: nil request")
	}
	if legacy := legacyForRequest(o); legacy != nil {
		return newClientCodec(conn, o, legacy), nil
	}
	cf, err := Lookup(o.Request.Header[headers.ContentType], o.Codecs)
	if err != nil {
		return nil, err
	}
	return newClientCodec(conn, o, cf), nil
}

// NewServer builds a framing codec that reads requests and writes responses.
// Its String() returns the wire protocol ("mucp" or "grpc").
func NewServer(conn Conn, o Options) (codec.Codec, error) {
	if o.Request == nil {
		return nil, errs.New("mucp: nil request")
	}
	if legacy := legacyForResponse(o); legacy != nil {
		return newServerCodec(conn, o, legacy), nil
	}
	cf, err := Lookup(o.Request.Header[headers.ContentType], o.Codecs)
	if err != nil {
		return nil, err
	}
	return newServerCodec(conn, o, cf), nil
}

// legacyForRequest resolves a legacy codec for an outbound (client) frame.
func legacyForRequest(o Options) codec.NewCodec {
	if len(o.Protocol) > 0 {
		return nil
	}
	if len(o.Request.Header[headers.Message]) > 0 {
		return nil
	}
	switch o.Request.Header[headers.ContentType] {
	case "application/json":
		o.Request.Header[headers.ContentType] = "application/json-rpc"
	case "application/protobuf":
		o.Request.Header[headers.ContentType] = "application/proto-rpc"
	}
	return legacyCodecs[o.Request.Header[headers.ContentType]]
}

// legacyForResponse resolves a legacy codec for an inbound (server) frame. It
// also backfills Method/Endpoint on the request headers so the router sees a
// consistent message.
func legacyForResponse(o Options) codec.NewCodec {
	h := o.Request.Header
	if len(GetHeader(headers.Protocol, h)) > 0 {
		return nil
	}
	if len(GetHeader(headers.Message, h)) > 0 {
		return nil
	}
	service := GetHeader(headers.Request, h)
	method := GetHeader(headers.Method, h)
	endpoint := GetHeader(headers.Endpoint, h)
	target := GetHeader(headers.Target, h)
	if (len(service) == 0 && len(method) == 0 && len(endpoint) == 0) || len(target) > 0 {
		return legacyCodecs[h[headers.ContentType]]
	}
	if len(method) == 0 {
		h[headers.Method] = endpoint
	}
	if len(endpoint) == 0 {
		h[headers.Endpoint] = method
	}
	return nil
}

// errWrap wraps transport/codec errors in the go-micro envelope.
func errWrap(domain, kind string, err error) error {
	if err == nil {
		return nil
	}
	if domain == "" {
		return err
	}
	return merrors.InternalServerError(domain+"."+kind, err.Error())
}

// readWriteCloser adapts the codec's io.ReadWriteCloser to two byte buffers.
type readWriteCloser struct {
	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
	sync.RWMutex
}

func (rwc *readWriteCloser) Read(p []byte) (int, error) {
	rwc.RLock()
	defer rwc.RUnlock()
	return rwc.rbuf.Read(p)
}

func (rwc *readWriteCloser) Write(p []byte) (int, error) {
	rwc.Lock()
	defer rwc.Unlock()
	return rwc.wbuf.Write(p)
}

func (rwc *readWriteCloser) Close() error { return nil }

// readHeaders pulls wire headers into the codec message. The server also
// captures Target and falls Endpoint back to Method for legacy frames.
func readHeaders(m *codec.Message, server bool) {
	set := func(v, hdr string) string {
		if len(v) > 0 {
			return v
		}
		return GetHeader(hdr, m.Header)
	}
	m.Id = set(m.Id, headers.ID)
	m.Error = set(m.Error, headers.Error)
	m.Endpoint = set(m.Endpoint, headers.Endpoint)
	m.Method = set(m.Method, headers.Method)
	if server {
		m.Target = set(m.Target, headers.Request)
		if len(m.Endpoint) == 0 {
			m.Endpoint = m.Method
		}
	}
}

// writeHeaders serializes m onto the wire. server also writes X- prefixed
// copies so pre-protocol readers still find their headers.
func writeHeaders(m *codec.Message, stream string, server bool) {
	set := func(hdr, v string) {
		if len(v) == 0 {
			return
		}
		m.Header[hdr] = v
		if server {
			m.Header["X-"+hdr] = v
		}
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

// clientCodec frames outgoing requests / incoming responses.
type clientCodec struct {
	conn   Conn
	codec  codec.Codec
	req    *transport.Message
	buf    *readWriteCloser
	stream string
	domain string
}

func newClientCodec(conn Conn, o Options, cf codec.NewCodec) codec.Codec {
	rwc := &readWriteCloser{wbuf: bytes.NewBuffer(nil), rbuf: bytes.NewBuffer(nil)}
	return &clientCodec{
		conn:   conn,
		codec:  cf(rwc),
		req:    o.Request,
		buf:    rwc,
		stream: o.Stream,
		domain: o.Domain,
	}
}

func (c *clientCodec) Write(m *codec.Message, body interface{}) error {
	c.buf.wbuf.Reset()

	if m.Header == nil {
		m.Header = map[string]string{}
	}
	for k, v := range c.req.Header {
		m.Header[k] = v
	}
	writeHeaders(m, c.stream, false)

	if body != nil {
		if b, ok := body.(*raw.Frame); ok {
			m.Body = b.Data
		} else {
			if err := c.codec.Write(m, body); err != nil {
				return errWrap(c.domain, "codec", err)
			}
			m.Body = c.buf.wbuf.Bytes()
		}
	}

	return errWrap(c.domain, "transport", c.conn.Send(&transport.Message{Header: m.Header, Body: m.Body}))
}

func (c *clientCodec) ReadHeader(m *codec.Message, r codec.MessageType) error {
	var tm transport.Message
	if err := c.conn.Recv(&tm); err != nil {
		return errWrap(c.domain, "transport", err)
	}

	c.buf.rbuf.Reset()
	c.buf.rbuf.Write(tm.Body)
	m.Header = tm.Header

	if err := c.codec.ReadHeader(m, r); err != nil {
		return errWrap(c.domain, "codec", err)
	}
	readHeaders(m, false)

	return nil
}

func (c *clientCodec) ReadBody(b interface{}) error {
	if v, ok := b.(*raw.Frame); ok {
		v.Data = c.buf.rbuf.Bytes()
		return nil
	}
	if err := c.codec.ReadBody(b); err != nil {
		return errWrap(c.domain, "codec", err)
	}
	return nil
}

func (c *clientCodec) Close() error {
	c.buf.Close()
	c.codec.Close()
	return errWrap(c.domain, "transport", c.conn.Close())
}

func (c *clientCodec) String() string { return "rpc" }

// serverCodec frames incoming requests / outgoing responses.
type serverCodec struct {
	conn     Conn
	codec    codec.Codec
	req      *transport.Message
	buf      *readWriteCloser
	first    chan bool
	protocol string
	sync.Mutex
}

func newServerCodec(conn Conn, o Options, cf codec.NewCodec) codec.Codec {
	rwc := &readWriteCloser{wbuf: bufferPool.Get(), rbuf: bufferPool.Get()}
	c := &serverCodec{
		conn:     conn,
		codec:    cf(rwc),
		req:      o.Request,
		buf:      rwc,
		first:    make(chan bool),
		protocol: "mucp",
	}
	// grpc frames ride an h2 body, so the first codec read must start from
	// the already-received request body.
	// ponytail: grpc preload hack, preserved for legacy peers; remove when
	// the grpc framing era is gated off.
	if c.codec.String() == "grpc" {
		rwc.rbuf.Write(o.Request.Body)
		c.protocol = "grpc"
	} else {
		close(c.first)
	}
	return c
}

func (c *serverCodec) ReadHeader(r *codec.Message, t codec.MessageType) error {
	// the initial message is pre-loaded from the accepted request
	mmsg := codec.Message{Header: c.req.Header, Body: c.req.Body}

	select {
	case <-c.first:
		// not the first message: read off the socket
		var tm transport.Message
		if err := c.conn.Recv(&tm); err != nil {
			return err
		}
		c.buf.rbuf.Reset()
		c.buf.rbuf.Write(tm.Body)
		mmsg.Header = tm.Header
		mmsg.Body = tm.Body
		c.req = &tm
	default:
		// lock to prevent a race on the first-message handoff; the channel
		// avoids a context switch on the hot path
		c.Lock()
		select {
		case <-c.first:
		default:
			close(c.first)
		}
		c.Unlock()
	}

	readHeaders(&mmsg, true)

	if err := c.codec.ReadHeader(&mmsg, codec.Request); err != nil {
		return err
	}

	*r = mmsg

	return nil
}

func (c *serverCodec) ReadBody(b interface{}) error {
	// don't read empty body
	if len(c.req.Body) == 0 {
		return nil
	}
	if v, ok := b.(*raw.Frame); ok {
		v.Data = c.req.Body
		return nil
	}
	return c.codec.ReadBody(b)
}

func (c *serverCodec) Write(r *codec.Message, b interface{}) error {
	c.buf.wbuf.Reset()

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
	writeHeaders(m, "", true)

	var body []byte

	if v, ok := b.(*raw.Frame); ok {
		body = v.Data
	} else if len(r.Body) > 0 {
		body = r.Body
	} else if err := c.codec.Write(m, b); err != nil {
		c.buf.wbuf.Reset()
		m.Error = pkgerrors.Wrapf(err, "Unable to encode body").Error()
		m.Header[headers.Error] = m.Error
		if err := c.codec.Write(m, nil); err != nil {
			return err
		}
	} else {
		body = c.buf.wbuf.Bytes()
	}

	if len(body) > 0 {
		m.Header["Content-Type"] = c.req.Header["Content-Type"]
	}

	return c.conn.Send(&transport.Message{Header: m.Header, Body: body})
}

func (c *serverCodec) Close() error {
	c.codec.Close()
	err := c.conn.Close()
	bufferPool.Put(c.buf.rbuf)
	bufferPool.Put(c.buf.wbuf)
	return err
}

func (c *serverCodec) String() string { return c.protocol }
