package grpc

import (
	"github.com/micro/go-micro/v2/client"
)

type grpcEvent struct {
	topic       string
	contentType string
	payload     interface{}
}

func newGRPCEvent(topic string, payload interface{}, contentType string, opts ...client.MessageOption) client.Message {
	var options client.MessageOptions
	for _, o := range opts {
		o(&options)
	}

	if len(options.ContentType) > 0 {
		contentType = options.ContentType
	}

	return &grpcEvent{
		payload:     payload,
		topic:       topic,
		contentType: contentType,
	}
}

func (g *grpcEvent) ContentType() string {
	return g.contentType
}

func (g *grpcEvent) Topic() string {
	return g.topic
}

func (g *grpcEvent) Payload() interface{} {
	return g.payload
}
