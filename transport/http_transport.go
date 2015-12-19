package transport

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"
)

type buffer struct {
	io.ReadWriter
}

type httpTransport struct{}

type httpTransportClient struct {
	ht       *httpTransport
	addr     string
	conn     net.Conn
	dialOpts dialOptions
	once     sync.Once

	sync.Mutex
	r    chan *http.Request
	bl   []*http.Request
	buff *bufio.Reader
}

type httpTransportSocket struct {
	r    chan *http.Request
	conn net.Conn
	once sync.Once

	sync.Mutex
	buff *bufio.Reader
}

type httpTransportListener struct {
	listener net.Listener
}

func (b *buffer) Close() error {
	return nil
}

func (h *httpTransportClient) Send(m *Message) error {
	header := make(http.Header)

	for k, v := range m.Header {
		header.Set(k, v)
	}

	reqB := bytes.NewBuffer(m.Body)
	defer reqB.Reset()
	buf := &buffer{
		reqB,
	}

	req := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "http",
			Host:   h.addr,
		},
		Header:        header,
		Body:          buf,
		ContentLength: int64(reqB.Len()),
		Host:          h.addr,
	}

	h.Lock()
	h.bl = append(h.bl, req)
	select {
	case h.r <- h.bl[0]:
		h.bl = h.bl[1:]
	default:
	}
	h.Unlock()

	return req.Write(h.conn)
}

func (h *httpTransportClient) Recv(m *Message) error {
	var r *http.Request
	if !h.dialOpts.stream {
		rc, ok := <-h.r
		if !ok {
			return io.EOF
		}
		r = rc
	}

	h.Lock()
	defer h.Unlock()
	if h.buff == nil {
		return io.EOF
	}

	rsp, err := http.ReadResponse(h.buff, r)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return err
	}

	mr := &Message{
		Header: make(map[string]string),
		Body:   b,
	}

	for k, v := range rsp.Header {
		if len(v) > 0 {
			mr.Header[k] = v[0]
		} else {
			mr.Header[k] = ""
		}
	}

	*m = *mr
	return nil
}

func (h *httpTransportClient) Close() error {
	err := h.conn.Close()
	h.once.Do(func() {
		h.Lock()
		h.buff.Reset(nil)
		h.buff = nil
		h.Unlock()
		close(h.r)
	})
	return err
}

func (h *httpTransportSocket) Recv(m *Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	r, err := http.ReadRequest(h.buff)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	r.Body.Close()

	mr := &Message{
		Header: make(map[string]string),
		Body:   b,
	}

	for k, v := range r.Header {
		if len(v) > 0 {
			mr.Header[k] = v[0]
		} else {
			mr.Header[k] = ""
		}
	}

	select {
	case h.r <- r:
	default:
	}

	*m = *mr
	return nil
}

func (h *httpTransportSocket) Send(m *Message) error {
	b := bytes.NewBuffer(m.Body)
	defer b.Reset()

	r := <-h.r

	rsp := &http.Response{
		Header:        r.Header,
		Body:          &buffer{b},
		Status:        "200 OK",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: int64(len(m.Body)),
	}

	for k, v := range m.Header {
		rsp.Header.Set(k, v)
	}

	select {
	case h.r <- r:
	default:
	}

	return rsp.Write(h.conn)
}

func (h *httpTransportSocket) error(m *Message) error {
	b := bytes.NewBuffer(m.Body)
	defer b.Reset()
	rsp := &http.Response{
		Header:        make(http.Header),
		Body:          &buffer{b},
		Status:        "500 Internal Server Error",
		StatusCode:    500,
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

func (h *httpTransportSocket) Close() error {
	err := h.conn.Close()
	h.once.Do(func() {
		h.Lock()
		h.buff.Reset(nil)
		h.buff = nil
		h.Unlock()
	})
	return err
}

func (h *httpTransportListener) Addr() string {
	return h.listener.Addr().String()
}

func (h *httpTransportListener) Close() error {
	return h.listener.Close()
}

func (h *httpTransportListener) Accept(fn func(Socket)) error {
	for {
		c, err := h.listener.Accept()
		if err != nil {
			return err
		}

		sock := &httpTransportSocket{
			conn: c,
			buff: bufio.NewReader(c),
			r:    make(chan *http.Request, 1),
		}

		go func() {
			// TODO: think of a better error response strategy
			defer func() {
				if r := recover(); r != nil {
					sock.Close()
				}
			}()

			fn(sock)
		}()
	}
	return nil
}

func (h *httpTransport) Dial(addr string, opts ...DialOption) (Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	var dopts dialOptions

	for _, opt := range opts {
		opt(&dopts)
	}

	return &httpTransportClient{
		ht:       h,
		addr:     addr,
		conn:     conn,
		buff:     bufio.NewReader(conn),
		dialOpts: dopts,
		r:        make(chan *http.Request, 1),
	}, nil
}

func (h *httpTransport) Listen(addr string) (Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &httpTransportListener{
		listener: l,
	}, nil
}

func (h *httpTransport) String() string {
	return "http"
}

func newHttpTransport(addrs []string, opt ...Option) *httpTransport {
	return &httpTransport{}
}
