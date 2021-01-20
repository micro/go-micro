package http

import (
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/codec"
)

type httpRequest struct {
	service     string
	method      string
	contentType string
	request     interface{}
	opts        client.RequestOptions
}

func newHTTPRequest(service, method string, request interface{}, contentType string, reqOpts ...client.RequestOption) client.Request {
	var opts client.RequestOptions
	for _, o := range reqOpts {
		o(&opts)
	}

	if len(opts.ContentType) > 0 {
		contentType = opts.ContentType
	}

	return &httpRequest{
		service:     service,
		method:      method,
		request:     request,
		contentType: contentType,
		opts:        opts,
	}
}

func (h *httpRequest) ContentType() string {
	return h.contentType
}

func (h *httpRequest) Service() string {
	return h.service
}

func (h *httpRequest) Method() string {
	return h.method
}

func (h *httpRequest) Endpoint() string {
	return h.method
}

func (h *httpRequest) Codec() codec.Writer {
	return nil
}

func (h *httpRequest) Body() interface{} {
	return h.request
}

func (h *httpRequest) Stream() bool {
	return h.opts.Stream
}
