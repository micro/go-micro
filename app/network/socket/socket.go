// Package socket provides a net socket network
package socket

import (
	"bufio"
	"crypto/tls"
	"encoding/gob"
	"errors"
	"net"
	"strings"
	"time"

	log "github.com/asim/nitro/app/logger"
	"github.com/asim/nitro/app/network"
	maddr "github.com/asim/nitro/util/addr"
	mnet "github.com/asim/nitro/util/net"
	mls "github.com/asim/nitro/util/tls"
)

type socketTransport struct {
	opts network.Options
}

type Client struct {
	dialOpts network.DialOptions
	conn     net.Conn
	enc      *gob.Encoder
	dec      *gob.Decoder
	encBuf   *bufio.Writer
	timeout  time.Duration
}

type Socket struct {
	conn    net.Conn
	enc     *gob.Encoder
	dec     *gob.Decoder
	encBuf  *bufio.Writer
	timeout time.Duration
}

type Listener struct {
	listener net.Listener
	timeout  time.Duration
}

func getNetwork(v string) (string, string) {
	if len(v) == 0 {
		return "tcp", v
	}

	parts := strings.Split(v, "://")
	if len(parts) > 1 {
		return parts[0], strings.Join(parts[1:], ":")
	}

	return "tcp", v
}

func (t *Client) Local() string {
	return t.conn.LocalAddr().String()
}

func (t *Client) Remote() string {
	return t.conn.RemoteAddr().String()
}

func (t *Client) Send(m *network.Message) error {
	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		t.conn.SetDeadline(time.Now().Add(t.timeout))
	}
	if err := t.enc.Encode(m); err != nil {
		return err
	}
	return t.encBuf.Flush()
}

func (t *Client) Recv(m *network.Message) error {
	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		t.conn.SetDeadline(time.Now().Add(t.timeout))
	}
	return t.dec.Decode(&m)
}

func (t *Client) Close() error {
	return t.conn.Close()
}

func (t *Socket) Local() string {
	return t.conn.LocalAddr().String()
}

func (t *Socket) Remote() string {
	return t.conn.RemoteAddr().String()
}

func (t *Socket) Recv(m *network.Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		t.conn.SetDeadline(time.Now().Add(t.timeout))
	}

	return t.dec.Decode(&m)
}

func (t *Socket) Send(m *network.Message) error {
	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		t.conn.SetDeadline(time.Now().Add(t.timeout))
	}
	if err := t.enc.Encode(m); err != nil {
		return err
	}
	return t.encBuf.Flush()
}

func (t *Socket) Close() error {
	return t.conn.Close()
}

func (t *Listener) Addr() string {
	return t.listener.Addr().String()
}

func (t *Listener) Close() error {
	return t.listener.Close()
}

func (t *Listener) Accept(fn func(network.Socket)) error {
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
		sock := &Socket{
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

func (t *socketTransport) Dial(addr string, opts ...network.DialOption) (network.Client, error) {
	dopts := network.DialOptions{
		Timeout: network.DefaultDialTimeout,
	}

	for _, opt := range opts {
		opt(&dopts)
	}

	var conn net.Conn
	var err error

	// get tcp, udp, ip network
	network, address := getNetwork(addr)

	// TODO: support dial option here rather than using internal config
	if t.opts.Secure || t.opts.TLSConfig != nil {
		config := t.opts.TLSConfig
		if config == nil {
			config = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: dopts.Timeout}, network, address, config)
	} else {
		conn, err = net.DialTimeout(network, address, dopts.Timeout)
	}

	if err != nil {
		return nil, err
	}

	encBuf := bufio.NewWriter(conn)

	return &Client{
		dialOpts: dopts,
		conn:     conn,
		encBuf:   encBuf,
		enc:      gob.NewEncoder(encBuf),
		dec:      gob.NewDecoder(conn),
		timeout:  t.opts.Timeout,
	}, nil
}

func (t *socketTransport) Listen(addr string, opts ...network.ListenOption) (network.Listener, error) {
	var options network.ListenOptions
	for _, o := range opts {
		o(&options)
	}

	var l net.Listener
	var err error

	// get tcp, udp, ip network
	network, address := getNetwork(addr)

	var fn func(addr string) (net.Listener, error)

	// TODO: support use of listen options
	if t.opts.Secure || t.opts.TLSConfig != nil {
		config := t.opts.TLSConfig

		fn = func(addr string) (net.Listener, error) {
			if config == nil {
				hosts := []string{address}

				// check if its a valid host:port
				if host, _, err := net.SplitHostPort(address); err == nil {
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
			return tls.Listen(network, address, config)
		}

	} else {
		fn = func(addr string) (net.Listener, error) {
			return net.Listen(network, addr)
		}
	}

	// don't both with port massaging with unix
	switch network {
	case "unix":
		l, err = fn(address)
	default:
		l, err = mnet.Listen(address, fn)
	}

	if err != nil {
		return nil, err
	}

	return &Listener{
		timeout:  t.opts.Timeout,
		listener: l,
	}, nil
}

func (t *socketTransport) Init(opts ...network.Option) error {
	for _, o := range opts {
		o(&t.opts)
	}
	return nil
}

func (t *socketTransport) Options() network.Options {
	return t.opts
}

func (t *socketTransport) String() string {
	return "socket"
}

func NewTransport(opts ...network.Option) network.Transport {
	var options network.Options
	for _, o := range opts {
		o(&options)
	}
	return &socketTransport{opts: options}
}
