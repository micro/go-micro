package http

import "github.com/asim/go-micro/v3/codec"

type httpMessage struct {
	topic       string
	payload     interface{}
	contentType string
	header      map[string]string
	body        []byte
	codec       codec.Reader
}

func (r *httpMessage) Topic() string {
	return r.topic
}

func (r *httpMessage) Payload() interface{} {
	return r.payload
}

func (r *httpMessage) ContentType() string {
	return r.contentType
}

func (r *httpMessage) Header() map[string]string {
	return r.header
}

func (r *httpMessage) Body() []byte {
	return r.body
}

func (r *httpMessage) Codec() codec.Reader {
	return r.codec
}
