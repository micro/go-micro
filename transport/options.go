package transport

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/micro/go-micro/codec"
)

type Options struct {
	Addrs     []string
	Codec     codec.Marshaler
	Secure    bool
	TLSConfig *tls.Config
	// Timeout sets the timeout for Send/Recv
	Timeout time.Duration
	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type DialOptions struct {
	Stream  bool
	Timeout time.Duration

	// TODO: add tls options when dialling
	// Currently set in global options

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type ListenOptions struct {
	// TODO: add tls options when listening
	// Currently set in global options

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

// Addrs to use for transport
func Addrs(addrs ...string) Option {
	return func(o *Options) {
		o.Addrs = addrs
	}
}

// Codec sets the codec used for encoding where the transport
// does not support message headers
func Codec(c codec.Marshaler) Option {
	return func(o *Options) {
		o.Codec = c
	}
}

// Timeout sets the timeout for Send/Recv execution
func Timeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}

// Use secure communication. If TLSConfig is not specified we
// use InsecureSkipVerify and generate a self signed cert
func Secure(b bool) Option {
	return func(o *Options) {
		o.Secure = b
	}
}

// TLSConfig to be used for the transport.
func TLSConfig(t *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = t
	}
}

// Indicates whether this is a streaming connection
func WithStream() DialOption {
	return func(o *DialOptions) {
		o.Stream = true
	}
}

// Timeout used when dialling the remote side
func WithTimeout(d time.Duration) DialOption {
	return func(o *DialOptions) {
		o.Timeout = d
	}
}
