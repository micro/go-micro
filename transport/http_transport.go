package transport

import (
	"bufio"
	"crypto/tls"
	"net"
	"net/http"

	"go-micro.dev/v5/logger"
	maddr "go-micro.dev/v5/util/addr"
	mnet "go-micro.dev/v5/util/net"
	mls "go-micro.dev/v5/util/tls"
)

type httpTransport struct {
	opts Options
}

func NewHTTPTransport(opts ...Option) *httpTransport {
	options := Options{
		BuffSizeH2: DefaultBufSizeH2,
		Logger:     logger.DefaultLogger,
	}

	for _, o := range opts {
		o(&options)
	}

	return &httpTransport{opts: options}
}

func (h *httpTransport) Init(opts ...Option) error {
	for _, o := range opts {
		o(&h.opts)
	}

	return nil
}

func (h *httpTransport) Dial(addr string, opts ...DialOption) (Client, error) {
	dopts := DialOptions{
		Timeout: DefaultDialTimeout,
	}

	for _, opt := range opts {
		opt(&dopts)
	}

	var (
		conn net.Conn
		err  error
	)

	if h.opts.Secure || h.opts.TLSConfig != nil {
		config := h.opts.TLSConfig
		if config == nil {
			config = &tls.Config{
				InsecureSkipVerify: dopts.InsecureSkipVerify,
			}
		}

		config.NextProtos = []string{"http/1.1"}

		conn, err = newConn(func(addr string) (net.Conn, error) {
			return tls.DialWithDialer(&net.Dialer{Timeout: dopts.Timeout}, "tcp", addr, config)
		})(addr)
	} else {
		conn, err = newConn(func(addr string) (net.Conn, error) {
			return net.DialTimeout("tcp", addr, dopts.Timeout)
		})(addr)
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
		req:      make(chan *http.Request, 100),
		local:    conn.LocalAddr().String(),
		remote:   conn.RemoteAddr().String(),
	}, nil
}

func (h *httpTransport) Listen(addr string, opts ...ListenOption) (Listener, error) {
	var options ListenOptions
	for _, o := range opts {
		o(&options)
	}

	var (
		list net.Listener
		err  error
	)

	switch listener := getNetListener(&options); {
	// Extracted listener from context
	case listener != nil:
		getList := func(addr string) (net.Listener, error) {
			return listener, nil
		}

		list, err = mnet.Listen(addr, getList)

	// Needs to create self signed certificate
	case h.opts.Secure || h.opts.TLSConfig != nil:
		config := h.opts.TLSConfig

		getList := func(addr string) (net.Listener, error) {
			if config != nil {
				return tls.Listen("tcp", addr, config)
			}

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

			config = &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			}

			return tls.Listen("tcp", addr, config)
		}

		list, err = mnet.Listen(addr, getList)

	// Create new basic net listener
	default:
		getList := func(addr string) (net.Listener, error) {
			return net.Listen("tcp", addr)
		}

		list, err = mnet.Listen(addr, getList)
	}

	if err != nil {
		return nil, err
	}

	return &httpTransportListener{
		ht:       h,
		listener: list,
	}, nil
}

func (h *httpTransport) Options() Options {
	return h.opts
}

func (h *httpTransport) String() string {
	return "http"
}
