package server

import (
	"io"
	"net/rpc"

	"github.com/youtube/vitess/go/rpcplus"
	"github.com/youtube/vitess/go/rpcplus/jsonrpc"
	"github.com/youtube/vitess/go/rpcplus/pbrpc"
)

var (
	defaultCodecs = map[string]codecFunc{
		"application/json":         jsonrpc.NewServerCodec,
		"application/json-rpc":     jsonrpc.NewServerCodec,
		"application/protobuf":     pbrpc.NewServerCodec,
		"application/proto-rpc":    pbrpc.NewServerCodec,
		"application/octet-stream": pbrpc.NewServerCodec,
	}
)

// CodecFunc is used to encode/decode requests/responses
type CodecFunc func(io.ReadWriteCloser) rpc.ServerCodec

// for internal use only
type codecFunc func(io.ReadWriteCloser) rpcplus.ServerCodec

// wraps an net/rpc ServerCodec to provide an rpcplus.ServerCodec
// temporary until we strip out use of rpcplus
type rpcCodecWrap struct {
	r rpc.ServerCodec
}

func (cw *rpcCodecWrap) ReadRequestHeader(r *rpcplus.Request) error {
	rc := &rpc.Request{
		ServiceMethod: r.ServiceMethod,
		Seq:           r.Seq,
	}
	err := cw.r.ReadRequestHeader(rc)
	r.ServiceMethod = rc.ServiceMethod
	r.Seq = rc.Seq
	return err
}

func (cw *rpcCodecWrap) ReadRequestBody(b interface{}) error {
	return cw.r.ReadRequestBody(b)
}

func (cw *rpcCodecWrap) WriteResponse(r *rpcplus.Response, b interface{}, l bool) error {
	rc := &rpc.Response{
		ServiceMethod: r.ServiceMethod,
		Seq:           r.Seq,
		Error:         r.Error,
	}
	err := cw.r.WriteResponse(rc, b)
	r.ServiceMethod = rc.ServiceMethod
	r.Seq = rc.Seq
	r.Error = r.Error
	return err
}

func (cw *rpcCodecWrap) Close() error {
	return cw.r.Close()
}

// wraps a CodecFunc to provide an internal codecFunc
// temporary until we strip rpcplus out
func codecWrap(cf CodecFunc) codecFunc {
	return func(rwc io.ReadWriteCloser) rpcplus.ServerCodec {
		return &rpcCodecWrap{
			r: cf(rwc),
		}
	}
}
