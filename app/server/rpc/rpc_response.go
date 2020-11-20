package rpc

import (
	"net/http"

	"github.com/asim/nitro/app/codec"
	"github.com/asim/nitro/app/network"
)

type rpcResponse struct {
	header map[string]string
	socket network.Socket
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
	if _, ok := r.header["Content-Type"]; !ok {
		r.header["Content-Type"] = http.DetectContentType(b)
	}

	return r.socket.Send(&network.Message{
		Header: r.header,
		Body:   b,
	})
}
