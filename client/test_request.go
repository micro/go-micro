package client

import (
	"github.com/micro/go-micro/v3/codec"
)

type testRequest struct {
	service     string
	method      string
	endpoint    string
	contentType string
	codec       codec.Codec
	body        interface{}
	opts        RequestOptions
}

func newRequest(service, endpoint string, request interface{}, contentType string, reqOpts ...RequestOption) Request {
	var opts RequestOptions

	for _, o := range reqOpts {
		o(&opts)
	}

	// set the content-type specified
	if len(opts.ContentType) > 0 {
		contentType = opts.ContentType
	}

	return &testRequest{
		service:     service,
		method:      endpoint,
		endpoint:    endpoint,
		body:        request,
		contentType: contentType,
		opts:        opts,
	}
}

func (r *testRequest) ContentType() string {
	return r.contentType
}

func (r *testRequest) Service() string {
	return r.service
}

func (r *testRequest) Method() string {
	return r.method
}

func (r *testRequest) Endpoint() string {
	return r.endpoint
}

func (r *testRequest) Body() interface{} {
	return r.body
}

func (r *testRequest) Codec() codec.Writer {
	return r.codec
}

func (r *testRequest) Stream() bool {
	return r.opts.Stream
}
