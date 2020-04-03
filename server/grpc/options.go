package grpc

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/broker"
	"github.com/micro/go-micro/v2/codec"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/server"
	"github.com/micro/go-micro/v2/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
)

type codecsKey struct{}
type grpcOptions struct{}
type netListener struct{}
type maxMsgSizeKey struct{}
type maxConnKey struct{}
type tlsAuth struct{}

// gRPC Codec to be used to encode/decode requests for a given content type
func Codec(contentType string, c encoding.Codec) server.Option {
	return func(o *server.Options) {
		codecs := make(map[string]encoding.Codec)
		if o.Context == nil {
			o.Context = context.Background()
		}
		if v, ok := o.Context.Value(codecsKey{}).(map[string]encoding.Codec); ok && v != nil {
			codecs = v
		}
		codecs[contentType] = c
		o.Context = context.WithValue(o.Context, codecsKey{}, codecs)
	}
}

// AuthTLS should be used to setup a secure authentication using TLS
func AuthTLS(t *tls.Config) server.Option {
	return setServerOption(tlsAuth{}, t)
}

// MaxConn specifies maximum number of max simultaneous connections to server
func MaxConn(n int) server.Option {
	return setServerOption(maxConnKey{}, n)
}

// Listener specifies the net.Listener to use instead of the default
func Listener(l net.Listener) server.Option {
	return setServerOption(netListener{}, l)
}

// Options to be used to configure gRPC options
func Options(opts ...grpc.ServerOption) server.Option {
	return setServerOption(grpcOptions{}, opts)
}

//
// MaxMsgSize set the maximum message in bytes the server can receive and
// send.  Default maximum message size is 4 MB.
//
func MaxMsgSize(s int) server.Option {
	return setServerOption(maxMsgSizeKey{}, s)
}

func newOptions(opt ...server.Option) server.Options {
	opts := server.Options{
		Auth:      auth.DefaultAuth,
		Codecs:    make(map[string]codec.NewCodec),
		Metadata:  map[string]string{},
		Broker:    broker.DefaultBroker,
		Registry:  registry.DefaultRegistry,
		Transport: transport.DefaultTransport,
		Address:   server.DefaultAddress,
		Name:      server.DefaultName,
		Id:        server.DefaultId,
		Version:   server.DefaultVersion,
	}

	for _, o := range opt {
		o(&opts)
	}

	return opts
}
