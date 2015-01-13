package client

import (
	"net/http"
)

type RpcRequest struct {
	service, method, contentType string
	request                      interface{}
	headers                      http.Header
}

func newRpcRequest(service, method string, request interface{}, contentType string) *RpcRequest {
	return &RpcRequest{
		service:     service,
		method:      method,
		request:     request,
		contentType: contentType,
		headers:     make(http.Header),
	}
}

func (r *RpcRequest) ContentType() string {
	return r.contentType
}

func (r *RpcRequest) Headers() Headers {
	return r.headers
}

func (r *RpcRequest) Service() string {
	return r.service
}

func (r *RpcRequest) Method() string {
	return r.method
}

func (r *RpcRequest) Request() interface{} {
	return r.request
}

func NewRpcRequest(service, method string, request interface{}, contentType string) *RpcRequest {
	return newRpcRequest(service, method, request, contentType)
}
