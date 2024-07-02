package client

import (
	"go-micro.dev/v5/codec"
	"go-micro.dev/v5/transport"
)

type rpcResponse struct {
	socket transport.Socket
	codec  codec.Codec
	header map[string]string
	body   []byte
}

func (r *rpcResponse) Codec() codec.Reader {
	return r.codec
}

func (r *rpcResponse) Header() map[string]string {
	return r.header
}

func (r *rpcResponse) Read() ([]byte, error) {
	var msg transport.Message

	if err := r.socket.Recv(&msg); err != nil {
		return nil, err
	}

	// set internals
	r.header = msg.Header
	r.body = msg.Body

	return msg.Body, nil
}
