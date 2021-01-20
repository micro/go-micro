package http

import (
	"github.com/asim/go-micro/v3/client"
)

type httpMessage struct {
	topic       string
	contentType string
	payload     interface{}
}

func newHTTPMessage(topic string, payload interface{}, contentType string, opts ...client.MessageOption) client.Message {
	var options client.MessageOptions
	for _, o := range opts {
		o(&options)
	}

	if len(options.ContentType) > 0 {
		contentType = options.ContentType
	}

	return &httpMessage{
		payload:     payload,
		topic:       topic,
		contentType: contentType,
	}
}

func (h *httpMessage) ContentType() string {
	return h.contentType
}

func (h *httpMessage) Topic() string {
	return h.topic
}

func (h *httpMessage) Payload() interface{} {
	return h.payload
}
