package mucp

import (
	"github.com/micro/go-micro/v3/client"
)

type message struct {
	topic       string
	contentType string
	payload     interface{}
}

func newMessage(topic string, payload interface{}, contentType string, opts ...client.MessageOption) client.Message {
	var options client.MessageOptions
	for _, o := range opts {
		o(&options)
	}

	if len(options.ContentType) > 0 {
		contentType = options.ContentType
	}

	return &message{
		payload:     payload,
		topic:       topic,
		contentType: contentType,
	}
}

func (m *message) ContentType() string {
	return m.contentType
}

func (m *message) Topic() string {
	return m.topic
}

func (m *message) Payload() interface{} {
	return m.payload
}
