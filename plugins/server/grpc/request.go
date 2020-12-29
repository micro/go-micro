package grpc

import (
	"github.com/micro/go-micro/v2/codec"
	"github.com/micro/go-micro/v2/codec/bytes"
)

type rpcRequest struct {
	service     string
	method      string
	contentType string
	codec       codec.Codec
	header      map[string]string
	body        []byte
	stream      bool
	payload     interface{}
}

type rpcMessage struct {
	topic       string
	contentType string
	payload     interface{}
	header      map[string]string
	body        []byte
	codec       codec.Codec
}

func (r *rpcRequest) ContentType() string {
	return r.contentType
}

func (r *rpcRequest) Service() string {
	return r.service
}

func (r *rpcRequest) Method() string {
	return r.method
}

func (r *rpcRequest) Endpoint() string {
	return r.method
}

func (r *rpcRequest) Codec() codec.Reader {
	return r.codec
}

func (r *rpcRequest) Header() map[string]string {
	return r.header
}

func (r *rpcRequest) Read() ([]byte, error) {
	f := &bytes.Frame{}
	if err := r.codec.ReadBody(f); err != nil {
		return nil, err
	}
	return f.Data, nil
}

func (r *rpcRequest) Stream() bool {
	return r.stream
}

func (r *rpcRequest) Body() interface{} {
	return r.payload
}

func (r *rpcMessage) ContentType() string {
	return r.contentType
}

func (r *rpcMessage) Topic() string {
	return r.topic
}

func (r *rpcMessage) Payload() interface{} {
	return r.payload
}

func (r *rpcMessage) Header() map[string]string {
	return r.header
}

func (r *rpcMessage) Body() []byte {
	return r.body
}

func (r *rpcMessage) Codec() codec.Reader {
	return r.codec
}
