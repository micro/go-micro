package grpc

import (
	"github.com/micro/go-micro/v2/codec"
)

type rpcResponse struct {
	header map[string]string
	codec  codec.Codec
}

func (r *rpcResponse) Codec() codec.Writer {
	return r.codec
}

func (r *rpcResponse) WriteHeader(hdr map[string]string) {
	for k, v := range hdr {
		r.header[k] = v
	}
}

func (r *rpcResponse) Write(b []byte) error {
	return r.codec.Write(&codec.Message{
		Header: r.header,
		Body:   b,
	}, nil)
}
