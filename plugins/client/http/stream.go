package http

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/asim/go-micro/v3/client"
)

// Implements the streamer interface
type httpStream struct {
	sync.RWMutex
	address string
	codec   Codec
	context context.Context
	header  http.Header
	seq     uint64
	closed  chan bool
	err     error
	conn    net.Conn
	reader  *bufio.Reader
	request client.Request
}

var (
	errShutdown = errors.New("connection is shut down")
)

func (h *httpStream) isClosed() bool {
	select {
	case <-h.closed:
		return true
	default:
		return false
	}
}

func (h *httpStream) Context() context.Context {
	return h.context
}

func (h *httpStream) Request() client.Request {
	return h.request
}

func (h *httpStream) Response() client.Response {
	return nil
}

func (h *httpStream) Send(msg interface{}) error {
	h.Lock()
	defer h.Unlock()

	if h.isClosed() {
		h.err = errShutdown
		return errShutdown
	}

	b, err := h.codec.Marshal(msg)
	if err != nil {
		return err
	}

	buf := &buffer{bytes.NewBuffer(b)}
	defer buf.Close()

	req := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "http",
			Host:   h.address,
			Path:   h.request.Endpoint(),
		},
		Header:        h.header,
		Body:          buf,
		ContentLength: int64(len(b)),
		Host:          h.address,
	}

	return req.Write(h.conn)
}

func (h *httpStream) Recv(msg interface{}) error {
	h.Lock()
	defer h.Unlock()

	if h.isClosed() {
		h.err = errShutdown
		return errShutdown
	}

	rsp, err := http.ReadResponse(h.reader, new(http.Request))
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return err
	}

	if rsp.StatusCode != 200 {
		return errors.New(rsp.Status + ": " + string(b))
	}

	return h.codec.Unmarshal(b, msg)
}

func (h *httpStream) Error() error {
	h.RLock()
	defer h.RUnlock()
	return h.err
}

func (h *httpStream) Close() error {
	select {
	case <-h.closed:
		return nil
	default:
		close(h.closed)
		return h.conn.Close()
	}
}
