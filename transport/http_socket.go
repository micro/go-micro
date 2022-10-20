package transport

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type httpTransportSocket struct {
	ht *httpTransport
	w  http.ResponseWriter
	r  *http.Request
	rw *bufio.ReadWriter

	mtx sync.RWMutex

	// the hijacked when using http 1
	conn net.Conn
	// for the first request
	ch chan *http.Request

	// h2 things
	buf *bufio.Reader
	// indicate if socket is closed
	closed chan bool

	// local/remote ip
	local  string
	remote string
}

func (h *httpTransportSocket) Local() string {
	return h.local
}

func (h *httpTransportSocket) Remote() string {
	return h.remote
}

func (h *httpTransportSocket) Recv(msg *Message) error {
	if msg == nil {
		return errors.New("message passed in is nil")
	}

	if msg.Header == nil {
		msg.Header = make(map[string]string, len(h.r.Header))
	}

	if h.r.ProtoMajor == 1 {
		return h.recvHTTP1(msg)
	}

	return h.recvHTTP2(msg)
}

func (h *httpTransportSocket) Send(msg *Message) error {
	// we need to lock to protect the write
	h.mtx.RLock()
	defer h.mtx.RUnlock()

	if h.r.ProtoMajor == 1 {
		return h.sendHTTP1(msg)
	}

	return h.sendHTTP2(msg)
}

func (h *httpTransportSocket) Close() error {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	select {
	case <-h.closed:
		return nil
	default:
		// Close the channel
		close(h.closed)

		// Close the buffer
		if err := h.r.Body.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (h *httpTransportSocket) error(m *Message) error {
	if h.r.ProtoMajor == 1 {
		rsp := &http.Response{
			Header:        make(http.Header),
			Body:          io.NopCloser(bytes.NewReader(m.Body)),
			Status:        "500 Internal Server Error",
			StatusCode:    http.StatusInternalServerError,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: int64(len(m.Body)),
		}

		for k, v := range m.Header {
			rsp.Header.Set(k, v)
		}

		return rsp.Write(h.conn)
	}

	return nil
}

func (h *httpTransportSocket) recvHTTP1(msg *Message) error {
	// set timeout if its greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		if err := h.conn.SetDeadline(time.Now().Add(h.ht.opts.Timeout)); err != nil {
			return errors.Wrap(err, "failed to set deadline")
		}
	}

	var req *http.Request

	select {
	// get first request
	case req = <-h.ch:
	// read next request
	default:
		rr, err := http.ReadRequest(h.rw.Reader)
		if err != nil {
			return errors.Wrap(err, "failed to read request")
		}

		req = rr
	}

	// read body
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read body")
	}

	// set body
	if err := req.Body.Close(); err != nil {
		return errors.Wrap(err, "failed to close body")
	}

	msg.Body = b

	// set headers
	for k, v := range req.Header {
		if len(v) > 0 {
			msg.Header[k] = v[0]
		} else {
			msg.Header[k] = ""
		}
	}

	// return early early
	return nil
}

func (h *httpTransportSocket) recvHTTP2(msg *Message) error {
	// only process if the socket is open
	select {
	case <-h.closed:
		return io.EOF
	default:
	}

	// read streaming body

	// set max buffer size
	s := h.ht.opts.BuffSizeH2
	if s == 0 {
		s = DefaultBufSizeH2
	}

	buf := make([]byte, s)

	// read the request body
	n, err := h.buf.Read(buf)
	// not an eof error
	if err != nil {
		return err
	}

	// check if we have data
	if n > 0 {
		msg.Body = buf[:n]
	}

	// set headers
	for k, v := range h.r.Header {
		if len(v) > 0 {
			msg.Header[k] = v[0]
		} else {
			msg.Header[k] = ""
		}
	}

	// set path
	msg.Header[":path"] = h.r.URL.Path

	return nil
}

func (h *httpTransportSocket) sendHTTP1(msg *Message) error {
	// make copy of header
	hdr := make(http.Header)
	for k, v := range h.r.Header {
		hdr[k] = v
	}

	rsp := &http.Response{
		Header:        hdr,
		Body:          io.NopCloser(bytes.NewReader(msg.Body)),
		Status:        "200 OK",
		StatusCode:    http.StatusOK,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: int64(len(msg.Body)),
	}

	for k, v := range msg.Header {
		rsp.Header.Set(k, v)
	}

	// set timeout if its greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		if err := h.conn.SetDeadline(time.Now().Add(h.ht.opts.Timeout)); err != nil {
			return err
		}
	}

	return rsp.Write(h.conn)
}

func (h *httpTransportSocket) sendHTTP2(msg *Message) error {
	// only process if the socket is open
	select {
	case <-h.closed:
		return io.EOF
	default:
	}

	// set headers
	for k, v := range msg.Header {
		h.w.Header().Set(k, v)
	}

	// write request
	_, err := h.w.Write(msg.Body)

	// flush the trailers
	h.w.(http.Flusher).Flush()

	return err
}
