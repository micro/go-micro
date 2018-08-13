package transport

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/micro/go-log"
	maddr "github.com/micro/util/go/lib/addr"
	mnet "github.com/micro/util/go/lib/net"
	mls "github.com/micro/util/go/lib/tls"
)

type buffer struct {
	io.ReadWriter
}

type httpTransport struct {
	opts Options
}

type httpTransportClient struct {
	ht       *httpTransport
	addr     string
	conn     net.Conn
	dialOpts DialOptions
	once     sync.Once

	sync.Mutex
	r    chan *http.Request
	bl   []*http.Request
	buff *bufio.Reader
}

type httpTransportSocket struct {
	ht   *httpTransport
	r    chan *http.Request
	conn net.Conn
	once sync.Once

	sync.Mutex
	buff *bufio.Reader
}

type httpTransportListener struct {
	ht       *httpTransport
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

	// set timeout if its greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		h.conn.SetDeadline(time.Now().Add(h.ht.opts.Timeout))
	}

	return req.Write(h.conn)
}

func (h *httpTransportClient) Recv(m *Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	var r *http.Request
	if !h.dialOpts.Stream {
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

	// set timeout if its greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		h.conn.SetDeadline(time.Now().Add(h.ht.opts.Timeout))
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

	if rsp.StatusCode != 200 {
		return errors.New(rsp.Status + ": " + string(b))
	}

	m.Body = b

	if m.Header == nil {
		m.Header = make(map[string]string)
	}

	for k, v := range rsp.Header {
		if len(v) > 0 {
			m.Header[k] = v[0]
		} else {
			m.Header[k] = ""
		}
	}

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

	// set timeout if its greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		h.conn.SetDeadline(time.Now().Add(h.ht.opts.Timeout))
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
	m.Body = b

	if m.Header == nil {
		m.Header = make(map[string]string)
	}

	for k, v := range r.Header {
		if len(v) > 0 {
			m.Header[k] = v[0]
		} else {
			m.Header[k] = ""
		}
	}

	select {
	case h.r <- r:
	default:
	}

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

	// set timeout if its greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		h.conn.SetDeadline(time.Now().Add(h.ht.opts.Timeout))
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
	var tempDelay time.Duration

	for {
		c, err := h.listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Logf("http: Accept error: %v; retrying in %v\n", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}

		sock := &httpTransportSocket{
			ht:   h.ht,
			conn: c,
			buff: bufio.NewReader(c),
			r:    make(chan *http.Request, 1),
		}

		go func() {
			// TODO: think of a better error response strategy
			defer func() {
				if r := recover(); r != nil {
					log.Log("panic recovered: ", r)
					sock.Close()
				}
			}()

			fn(sock)
		}()
	}
}

func (h *httpTransport) Dial(addr string, opts ...DialOption) (Client, error) {
	dopts := DialOptions{
		Timeout: DefaultDialTimeout,
	}

	for _, opt := range opts {
		opt(&dopts)
	}

	var conn net.Conn
	var err error

	// TODO: support dial option here rather than using internal config
	if h.opts.Secure || h.opts.TLSConfig != nil {
		config := h.opts.TLSConfig
		if config == nil {
			config = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: dopts.Timeout}, "tcp", addr, config)
	} else {
		conn, err = net.DialTimeout("tcp", addr, dopts.Timeout)
	}

	if err != nil {
		return nil, err
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

func (h *httpTransport) Listen(addr string, opts ...ListenOption) (Listener, error) {
	var options ListenOptions
	for _, o := range opts {
		o(&options)
	}

	var l net.Listener
	var err error

	// TODO: support use of listen options
	if h.opts.Secure || h.opts.TLSConfig != nil {
		config := h.opts.TLSConfig

		fn := func(addr string) (net.Listener, error) {
			if config == nil {
				hosts := []string{addr}

				// check if its a valid host:port
				if host, _, err := net.SplitHostPort(addr); err == nil {
					if len(host) == 0 {
						hosts = maddr.IPs()
					} else {
						hosts = []string{host}
					}
				}

				// generate a certificate
				cert, err := mls.Certificate(hosts...)
				if err != nil {
					return nil, err
				}
				config = &tls.Config{Certificates: []tls.Certificate{cert}}
			}
			return tls.Listen("tcp", addr, config)
		}

		l, err = mnet.Listen(addr, fn)
	} else {
		fn := func(addr string) (net.Listener, error) {
			return net.Listen("tcp", addr)
		}

		l, err = mnet.Listen(addr, fn)
	}

	if err != nil {
		return nil, err
	}

	return &httpTransportListener{
		ht:       h,
		listener: l,
	}, nil
}

func (h *httpTransport) Init(opts ...Option) error {
	for _, o := range opts {
		o(&h.opts)
	}
	return nil
}

func (h *httpTransport) Options() Options {
	return h.opts
}

func (h *httpTransport) String() string {
	return "http"
}

func newHTTPTransport(opts ...Option) *httpTransport {
	var options Options
	for _, o := range opts {
		o(&options)
	}
	return &httpTransport{opts: options}
}
