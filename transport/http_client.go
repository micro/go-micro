package transport

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/pkg/errors"

	log "go-micro.dev/v5/logger"
	"go-micro.dev/v5/util/buf"
)

type httpTransportClient struct {
	dialOpts DialOptions
	conn     net.Conn
	ht       *httpTransport

	// request must be stored for response processing
	req  chan *http.Request
	buff *bufio.Reader
	addr string

	// local/remote ip
	local   string
	remote  string
	reqList []*http.Request

	sync.RWMutex

	once sync.Once

	closed bool
}

func (h *httpTransportClient) Local() string {
	return h.local
}

func (h *httpTransportClient) Remote() string {
	return h.remote
}

func (h *httpTransportClient) Send(m *Message) error {
	logger := h.ht.Options().Logger

	header := make(http.Header)
	for k, v := range m.Header {
		header.Set(k, v)
	}

	b := buf.New(bytes.NewBuffer(m.Body))
	defer func() {
		if err := b.Close(); err != nil {
			logger.Logf(log.ErrorLevel, "failed to close buffer: %v", err)
		}
	}()

	req := &http.Request{
		Method: http.MethodPost,
		URL: &url.URL{
			Scheme: "http",
			Host:   h.addr,
		},
		Header:        header,
		Body:          b,
		ContentLength: int64(b.Len()),
		Host:          h.addr,
		Close:         h.dialOpts.ConnClose,
	}

	if !h.dialOpts.Stream {
		h.Lock()
		if h.closed {
			h.Unlock()
			return io.EOF
		}

		h.reqList = append(h.reqList, req)

		select {
		case h.req <- h.reqList[0]:
			h.reqList = h.reqList[1:]
		default:
		}
		h.Unlock()
	}

	// set timeout if its greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		if err := h.conn.SetDeadline(time.Now().Add(h.ht.opts.Timeout)); err != nil {
			return err
		}
	}

	return req.Write(h.conn)

}

// Recv receives a message.
func (h *httpTransportClient) Recv(msg *Message) (err error) {
	if msg == nil {
		return errors.New("message passed in is nil")
	}

	var req *http.Request

	if !h.dialOpts.Stream {
		rc, ok := <-h.req
		if !ok {
			h.Lock()
			if len(h.reqList) == 0 {
				h.Unlock()
				return io.EOF
			}

			rc = h.reqList[0]
			h.reqList = h.reqList[1:]
			h.Unlock()
		}

		req = rc
	}

	// set timeout if its greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		if err = h.conn.SetDeadline(time.Now().Add(h.ht.opts.Timeout)); err != nil {
			return err
		}
	}

	h.Lock()
	defer h.Unlock()

	if h.closed {
		return io.EOF
	}

	rsp, err := http.ReadResponse(h.buff, req)
	if err != nil {
		return err
	}

	defer func() {
		if err2 := rsp.Body.Close(); err2 != nil {
			err = errors.Wrap(err2, "failed to close body")
		}
	}()

	b, err := io.ReadAll(rsp.Body)
	if err != nil {
		return err
	}

	if rsp.StatusCode != http.StatusOK {
		return errors.New(rsp.Status + ": " + string(b))
	}

	msg.Body = b

	if msg.Header == nil {
		msg.Header = make(map[string]string, len(rsp.Header))
	}

	for k, v := range rsp.Header {
		if len(v) > 0 {
			msg.Header[k] = v[0]
		} else {
			msg.Header[k] = ""
		}
	}

	return nil
}

func (h *httpTransportClient) Close() error {
	if !h.dialOpts.Stream {
		h.once.Do(
			func() {
				h.Lock()
				h.buff.Reset(nil)
				h.closed = true
				h.Unlock()
				close(h.req)
			},
		)

		return h.conn.Close()
	}

	err := h.conn.Close()
	h.once.Do(
		func() {
			h.Lock()
			h.buff.Reset(nil)
			h.closed = true
			h.Unlock()
			close(h.req)
		},
	)

	return err
}
