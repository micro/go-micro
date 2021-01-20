// Package tcp provides a TCP transport
package tcp

import (
	"bufio"
	"crypto/tls"
	"encoding/gob"
	"errors"
	"net"
	"time"

	"github.com/asim/go-micro/v3/cmd"
	log "github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/transport"
	maddr "github.com/asim/go-micro/v3/util/addr"
	mnet "github.com/asim/go-micro/v3/util/net"
	mls "github.com/asim/go-micro/v3/util/tls"
)

type tcpTransport struct {
	opts transport.Options
}

type tcpTransportClient struct {
	dialOpts transport.DialOptions
	conn     net.Conn
	enc      *gob.Encoder
	dec      *gob.Decoder
	encBuf   *bufio.Writer
	timeout  time.Duration
}

type tcpTransportSocket struct {
	conn    net.Conn
	enc     *gob.Encoder
	dec     *gob.Decoder
	encBuf  *bufio.Writer
	timeout time.Duration
}

type tcpTransportListener struct {
	listener net.Listener
	timeout  time.Duration
}

func init() {
	cmd.DefaultTransports["tcp"] = NewTransport
}

func (t *tcpTransportClient) Local() string {
	return t.conn.LocalAddr().String()
}

func (t *tcpTransportClient) Remote() string {
	return t.conn.RemoteAddr().String()
}

func (t *tcpTransportClient) Send(m *transport.Message) error {
	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		t.conn.SetDeadline(time.Now().Add(t.timeout))
	}
	if err := t.enc.Encode(m); err != nil {
		return err
	}
	return t.encBuf.Flush()
}

func (t *tcpTransportClient) Recv(m *transport.Message) error {
	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		t.conn.SetDeadline(time.Now().Add(t.timeout))
	}
	return t.dec.Decode(&m)
}

func (t *tcpTransportClient) Close() error {
	return t.conn.Close()
}

func (t *tcpTransportSocket) Local() string {
	return t.conn.LocalAddr().String()
}

func (t *tcpTransportSocket) Remote() string {
	return t.conn.RemoteAddr().String()
}

func (t *tcpTransportSocket) Recv(m *transport.Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		t.conn.SetDeadline(time.Now().Add(t.timeout))
	}

	return t.dec.Decode(&m)
}

func (t *tcpTransportSocket) Send(m *transport.Message) error {
	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		t.conn.SetDeadline(time.Now().Add(t.timeout))
	}
	if err := t.enc.Encode(m); err != nil {
		return err
	}
	return t.encBuf.Flush()
}

func (t *tcpTransportSocket) Close() error {
	return t.conn.Close()
}

func (t *tcpTransportListener) Addr() string {
	return t.listener.Addr().String()
}

func (t *tcpTransportListener) Close() error {
	return t.listener.Close()
}

func (t *tcpTransportListener) Accept(fn func(transport.Socket)) error {
	var tempDelay time.Duration

	for {
		c, err := t.listener.Accept()
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
				log.Errorf("http: Accept error: %v; retrying in %v\n", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}

		encBuf := bufio.NewWriter(c)
		sock := &tcpTransportSocket{
			timeout: t.timeout,
			conn:    c,
			encBuf:  encBuf,
			enc:     gob.NewEncoder(encBuf),
			dec:     gob.NewDecoder(c),
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
}

func (t *tcpTransport) Dial(addr string, opts ...transport.DialOption) (transport.Client, error) {
	dopts := transport.DialOptions{
		Timeout: transport.DefaultDialTimeout,
	}

	for _, opt := range opts {
		opt(&dopts)
	}

	var conn net.Conn
	var err error

	// TODO: support dial option here rather than using internal config
	if t.opts.Secure || t.opts.TLSConfig != nil {
		config := t.opts.TLSConfig
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

	encBuf := bufio.NewWriter(conn)

	return &tcpTransportClient{
		dialOpts: dopts,
		conn:     conn,
		encBuf:   encBuf,
		enc:      gob.NewEncoder(encBuf),
		dec:      gob.NewDecoder(conn),
		timeout:  t.opts.Timeout,
	}, nil
}

func (t *tcpTransport) Listen(addr string, opts ...transport.ListenOption) (transport.Listener, error) {
	var options transport.ListenOptions
	for _, o := range opts {
		o(&options)
	}

	var l net.Listener
	var err error

	// TODO: support use of listen options
	if t.opts.Secure || t.opts.TLSConfig != nil {
		config := t.opts.TLSConfig

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

	return &tcpTransportListener{
		timeout:  t.opts.Timeout,
		listener: l,
	}, nil
}

func (t *tcpTransport) Init(opts ...transport.Option) error {
	for _, o := range opts {
		o(&t.opts)
	}
	return nil
}

func (t *tcpTransport) Options() transport.Options {
	return t.opts
}

func (t *tcpTransport) String() string {
	return "tcp"
}

func NewTransport(opts ...transport.Option) transport.Transport {
	var options transport.Options
	for _, o := range opts {
		o(&options)
	}
	return &tcpTransport{opts: options}
}
