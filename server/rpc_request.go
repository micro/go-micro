package server

import (
	"github.com/micro/go-micro/codec"
)

type rpcRequest struct {
	service     string
	method      string
	contentType string
	codec       codec.Codec
	body        []byte
	stream      bool
}

type rpcMessage struct {
	topic       string
	contentType string
	payload     interface{}
}

func (r *rpcRequest) Codec() codec.Codec {
	return r.codec
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

func (r *rpcRequest) Body() []byte {
	return r.body
}

func (r *rpcRequest) Stream() bool {
	return r.stream
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
