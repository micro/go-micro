package client

import (
	"context"
	"testing"
	"time"

	"github.com/asim/nitro/v3/codec"
)

func TestBackoff(t *testing.T) {
	results := []time.Duration{
		0 * time.Second,
		100 * time.Millisecond,
		600 * time.Millisecond,
		1900 * time.Millisecond,
		4300 * time.Millisecond,
		7900 * time.Millisecond,
	}

	r := &testRequest{
		service: "test",
		method:  "test",
	}

	for i := 0; i < 5; i++ {
		d, err := exponentialBackoff(context.TODO(), r, i)
		if err != nil {
			t.Fatal(err)
		}

		if d != results[i] {
			t.Fatalf("Expected equal than %v, got %v", results[i], d)
		}
	}
}

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
