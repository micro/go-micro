// Package grpc provides a grpc transport
package grpc

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/micro/go-micro/v2/transport"
	maddr "github.com/micro/go-micro/v2/util/addr"
	mnet "github.com/micro/go-micro/v2/util/net"
	mls "github.com/micro/go-micro/v2/util/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/micro/go-micro/v2/transport/grpc/proto"
)

type grpcTransport struct {
	opts transport.Options
}

type grpcTransportListener struct {
	listener net.Listener
	secure   bool
	tls      *tls.Config
}

func getTLSConfig(addr string) (*tls.Config, error) {
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

	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}

func (t *grpcTransportListener) Addr() string {
	return t.listener.Addr().String()
}

func (t *grpcTransportListener) Close() error {
	return t.listener.Close()
}

func (t *grpcTransportListener) Accept(fn func(transport.Socket)) error {
	var opts []grpc.ServerOption

	// setup tls if specified
	if t.secure || t.tls != nil {
		config := t.tls
		if config == nil {
			var err error
			addr := t.listener.Addr().String()
			config, err = getTLSConfig(addr)
			if err != nil {
				return err
			}
		}

		creds := credentials.NewTLS(config)
		opts = append(opts, grpc.Creds(creds))
	}

	// new service
	srv := grpc.NewServer(opts...)

	// register service
	pb.RegisterTransportServer(srv, &microTransport{addr: t.listener.Addr().String(), fn: fn})

	// start serving
	return srv.Serve(t.listener)
}

func (t *grpcTransport) Dial(addr string, opts ...transport.DialOption) (transport.Client, error) {
	dopts := transport.DialOptions{
		Timeout: transport.DefaultDialTimeout,
	}

	for _, opt := range opts {
		opt(&dopts)
	}

	options := []grpc.DialOption{
		grpc.WithTimeout(dopts.Timeout),
	}

	if t.opts.Secure || t.opts.TLSConfig != nil {
		config := t.opts.TLSConfig
		if config == nil {
			config = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		creds := credentials.NewTLS(config)
		options = append(options, grpc.WithTransportCredentials(creds))
	} else {
		options = append(options, grpc.WithInsecure())
	}

	// dial the server
	conn, err := grpc.Dial(addr, options...)
	if err != nil {
		return nil, err
	}

	// create stream
	stream, err := pb.NewTransportClient(conn).Stream(context.Background())
	if err != nil {
		return nil, err
	}

	// return a client
	return &grpcTransportClient{
		conn:   conn,
		stream: stream,
		local:  "localhost",
		remote: addr,
	}, nil
}

func (t *grpcTransport) Listen(addr string, opts ...transport.ListenOption) (transport.Listener, error) {
	var options transport.ListenOptions
	for _, o := range opts {
		o(&options)
	}

	ln, err := mnet.Listen(addr, func(addr string) (net.Listener, error) {
		return net.Listen("tcp", addr)
	})
	if err != nil {
		return nil, err
	}

	return &grpcTransportListener{
		listener: ln,
		tls:      t.opts.TLSConfig,
		secure:   t.opts.Secure,
	}, nil
}

func (t *grpcTransport) Init(opts ...transport.Option) error {
	for _, o := range opts {
		o(&t.opts)
	}
	return nil
}

func (t *grpcTransport) Options() transport.Options {
	return t.opts
}

func (t *grpcTransport) String() string {
	return "grpc"
}

func NewTransport(opts ...transport.Option) transport.Transport {
	var options transport.Options
	for _, o := range opts {
		o(&options)
	}
	return &grpcTransport{opts: options}
}
