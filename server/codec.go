package server

import (
	"io"
	"net/rpc"
	"sync"

	"github.com/micro/go-micro/codec"

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

// for internal use only
type codecFunc func(io.ReadWriteCloser) rpcplus.ServerCodec

// wraps an net/rpc ServerCodec to provide an rpcplus.ServerCodec
// temporary until we strip out use of rpcplus
type rpcCodecWrap struct {
	sync.Mutex
	rwc io.ReadWriteCloser
	c   codec.Codec
}

func (cw *rpcCodecWrap) ReadRequestHeader(r *rpcplus.Request) error {
	data, err := pbrpc.ReadNetString(cw.rwc)
	if err != nil {
		return err
	}
	rtmp := new(rpc.Request)
	err = cw.c.Unmarshal(data, rtmp)
	if err != nil {
		return err
	}
	r.ServiceMethod = rtmp.ServiceMethod
	r.Seq = rtmp.Seq
	return nil
}

func (cw *rpcCodecWrap) ReadRequestBody(b interface{}) error {
	data, err := pbrpc.ReadNetString(cw.rwc)
	if err != nil {
		return err
	}
	if b != nil {
		return cw.c.Unmarshal(data, b)
	}
	return nil
}

func (cw *rpcCodecWrap) WriteResponse(r *rpcplus.Response, b interface{}, l bool) error {
	cw.Lock()
	defer cw.Unlock()
	rtmp := &rpc.Response{ServiceMethod: r.ServiceMethod, Seq: r.Seq, Error: r.Error}
	data, err := cw.c.Marshal(rtmp)
	if err != nil {
		return err
	}
	_, err = pbrpc.WriteNetString(cw.rwc, data)
	if err != nil {
		return err
	}
	data, err = cw.c.Marshal(b)
	if err != nil {
		return err
	}
	_, err = pbrpc.WriteNetString(cw.rwc, data)
	if err != nil {
		return err
	}
	return nil
}

func (cw *rpcCodecWrap) Close() error {
	return cw.rwc.Close()
}

// wraps a CodecFunc to provide an internal codecFunc
// temporary until we strip rpcplus out
func codecWrap(c codec.Codec) codecFunc {
	return func(rwc io.ReadWriteCloser) rpcplus.ServerCodec {
		return &rpcCodecWrap{
			rwc: rwc,
			c:   c,
		}
	}
}
