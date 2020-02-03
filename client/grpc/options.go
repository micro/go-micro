// Package grpc provides a gRPC options
package grpc

import (
	"context"
	"crypto/tls"

	"github.com/micro/go-micro/v2/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
)

var (
	// DefaultPoolMaxStreams maximum streams on a connectioin
	// (20)
	DefaultPoolMaxStreams = 20

	// DefaultPoolMaxIdle maximum idle conns of a pool
	// (50)
	DefaultPoolMaxIdle = 50

	// DefaultMaxRecvMsgSize maximum message that client can receive
	// (4 MB).
	DefaultMaxRecvMsgSize = 1024 * 1024 * 4

	// DefaultMaxSendMsgSize maximum message that client can send
	// (4 MB).
	DefaultMaxSendMsgSize = 1024 * 1024 * 4
)

type poolMaxStreams struct{}
type poolMaxIdle struct{}
type codecsKey struct{}
type tlsAuth struct{}
type maxRecvMsgSizeKey struct{}
type maxSendMsgSizeKey struct{}
type grpcDialOptions struct{}
type grpcCallOptions struct{}

// maximum streams on a connectioin
func PoolMaxStreams(n int) client.Option {
	return func(o *client.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, poolMaxStreams{}, n)
	}
}

// maximum idle conns of a pool
func PoolMaxIdle(d int) client.Option {
	return func(o *client.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, poolMaxIdle{}, d)
	}
}

// gRPC Codec to be used to encode/decode requests for a given content type
func Codec(contentType string, c encoding.Codec) client.Option {
	return func(o *client.Options) {
		codecs := make(map[string]encoding.Codec)
		if o.Context == nil {
			o.Context = context.Background()
		}
		if v := o.Context.Value(codecsKey{}); v != nil {
			codecs = v.(map[string]encoding.Codec)
		}
		codecs[contentType] = c
		o.Context = context.WithValue(o.Context, codecsKey{}, codecs)
	}
}

// AuthTLS should be used to setup a secure authentication using TLS
func AuthTLS(t *tls.Config) client.Option {
	return func(o *client.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, tlsAuth{}, t)
	}
}

//
// MaxRecvMsgSize set the maximum size of message that client can receive.
//
func MaxRecvMsgSize(s int) client.Option {
	return func(o *client.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, maxRecvMsgSizeKey{}, s)
	}
}

//
// MaxSendMsgSize set the maximum size of message that client can send.
//
func MaxSendMsgSize(s int) client.Option {
	return func(o *client.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, maxSendMsgSizeKey{}, s)
	}
}

//
// DialOptions to be used to configure gRPC dial options
//
func DialOptions(opts ...grpc.DialOption) client.CallOption {
	return func(o *client.CallOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, grpcDialOptions{}, opts)
	}
}

//
// CallOptions to be used to configure gRPC call options
//
func CallOptions(opts ...grpc.CallOption) client.CallOption {
	return func(o *client.CallOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, grpcCallOptions{}, opts)
	}
}
