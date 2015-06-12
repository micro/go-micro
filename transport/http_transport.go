package transport

import (
	"bufio"
	"bytes"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
)

type httpTransport struct{}

type httpTransportClient struct {
	ht       *httpTransport
	addr     string
	conn     net.Conn
	buff     *bufio.Reader
	dialOpts dialOptions
	r        chan *http.Request
}

type httpTransportSocket struct {
	r    *http.Request
	conn net.Conn
}

type httpTransportListener struct {
	listener net.Listener
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

	h.r <- req

	return req.Write(h.conn)
}

func (h *httpTransportClient) Recv(m *Message) error {
	var r *http.Request
	if !h.dialOpts.stream {
		r = <-h.r
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
	return h.conn.Close()
}

func (h *httpTransportSocket) Recv(m *Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	b, err := ioutil.ReadAll(h.r.Body)
	if err != nil {
		return err
	}
	h.r.Body.Close()
	mr := &Message{
		Header: make(map[string]string),
		Body:   b,
	}

	for k, v := range h.r.Header {
		if len(v) > 0 {
			mr.Header[k] = v[0]
		} else {
			mr.Header[k] = ""
		}
	}

	*m = *mr
	return nil
}

func (h *httpTransportSocket) Send(m *Message) error {
	b := bytes.NewBuffer(m.Body)
	defer b.Reset()
	rsp := &http.Response{
		Header:        h.r.Header,
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

	return rsp.Write(h.conn)
}

func (h *httpTransportSocket) Close() error {
	return h.conn.Close()
}

func (h *httpTransportListener) Addr() string {
	return h.listener.Addr().String()
}

func (h *httpTransportListener) Close() error {
	return h.listener.Close()
}

func (h *httpTransportListener) Accept(fn func(Socket)) error {
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				return
			}

			fn(&httpTransportSocket{
				conn: conn,
				r:    r,
			})
		}),
	}

	return srv.Serve(h.listener)
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

func newHttpTransport(addrs []string, opt ...Option) *httpTransport {
	return &httpTransport{}
}
