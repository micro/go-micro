package client

import (
	"io"
	"net/rpc"

	"github.com/youtube/vitess/go/rpcplus"
	"github.com/youtube/vitess/go/rpcplus/jsonrpc"
	"github.com/youtube/vitess/go/rpcplus/pbrpc"
)

var (
	defaultContentType = "application/octet-stream"

	defaultCodecs = map[string]codecFunc{
		"application/json":         jsonrpc.NewClientCodec,
		"application/json-rpc":     jsonrpc.NewClientCodec,
		"application/protobuf":     pbrpc.NewClientCodec,
		"application/proto-rpc":    pbrpc.NewClientCodec,
		"application/octet-stream": pbrpc.NewClientCodec,
	}
)

type CodecFunc func(io.ReadWriteCloser) rpc.ClientCodec

// only for internal use
type codecFunc func(io.ReadWriteCloser) rpcplus.ClientCodec

// wraps an net/rpc ClientCodec to provide an rpcplus.ClientCodec
// temporary until we strip out use of rpcplus
type rpcCodecWrap struct {
	r rpc.ClientCodec
}

func (cw *rpcCodecWrap) WriteRequest(r *rpcplus.Request, b interface{}) error {
	rc := &rpc.Request{
		ServiceMethod: r.ServiceMethod,
		Seq:           r.Seq,
	}
	err := cw.r.WriteRequest(rc, b)
	r.ServiceMethod = rc.ServiceMethod
	r.Seq = rc.Seq
	return err
}

func (cw *rpcCodecWrap) ReadResponseHeader(r *rpcplus.Response) error {
	rc := &rpc.Response{
		ServiceMethod: r.ServiceMethod,
		Seq:           r.Seq,
		Error:         r.Error,
	}
	err := cw.r.ReadResponseHeader(rc)
	r.ServiceMethod = rc.ServiceMethod
	r.Seq = rc.Seq
	r.Error = r.Error
	return err
}

func (cw *rpcCodecWrap) ReadResponseBody(b interface{}) error {
	return cw.r.ReadResponseBody(b)
}

func (cw *rpcCodecWrap) Close() error {
	return cw.r.Close()
}

// wraps a CodecFunc to provide an internal codecFunc
// temporary until we strip rpcplus out
func codecWrap(cf CodecFunc) codecFunc {
	return func(rwc io.ReadWriteCloser) rpcplus.ClientCodec {
		return &rpcCodecWrap{
			r: cf(rwc),
		}
	}
}
