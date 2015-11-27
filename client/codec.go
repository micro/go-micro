package client

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
	defaultContentType = "application/octet-stream"

	defaultCodecs = map[string]codecFunc{
		"application/json":         jsonrpc.NewClientCodec,
		"application/json-rpc":     jsonrpc.NewClientCodec,
		"application/protobuf":     pbrpc.NewClientCodec,
		"application/proto-rpc":    pbrpc.NewClientCodec,
		"application/octet-stream": pbrpc.NewClientCodec,
	}
)

// only for internal use
type codecFunc func(io.ReadWriteCloser) rpcplus.ClientCodec

// wraps an net/rpc ClientCodec to provide an rpcplus.ClientCodec
// temporary until we strip out use of rpcplus
type rpcCodecWrap struct {
	sync.Mutex
	c   codec.Codec
	rwc io.ReadWriteCloser
}

func (cw *rpcCodecWrap) WriteRequest(r *rpcplus.Request, b interface{}) error {
	cw.Lock()
	defer cw.Unlock()
	req := &rpc.Request{ServiceMethod: r.ServiceMethod, Seq: r.Seq}
	data, err := cw.c.Marshal(req)
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

func (cw *rpcCodecWrap) ReadResponseHeader(r *rpcplus.Response) error {
	data, err := pbrpc.ReadNetString(cw.rwc)
	if err != nil {
		return err
	}
	rtmp := new(rpc.Response)
	err = cw.c.Unmarshal(data, rtmp)
	if err != nil {
		return err
	}
	r.ServiceMethod = rtmp.ServiceMethod
	r.Seq = rtmp.Seq
	r.Error = rtmp.Error
	return nil
}

func (cw *rpcCodecWrap) ReadResponseBody(b interface{}) error {
	data, err := pbrpc.ReadNetString(cw.rwc)
	if err != nil {
		return err
	}
	if b != nil {
		return cw.c.Unmarshal(data, b)
	}
	return nil
}

func (cw *rpcCodecWrap) Close() error {
	return cw.rwc.Close()
}

// wraps a CodecFunc to provide an internal codecFunc
// temporary until we strip rpcplus out
func codecWrap(c codec.Codec) codecFunc {
	return func(rwc io.ReadWriteCloser) rpcplus.ClientCodec {
		return &rpcCodecWrap{
			rwc: rwc,
			c:   c,
		}
	}
}
