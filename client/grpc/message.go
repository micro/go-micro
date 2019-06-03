package grpc

import (
	"github.com/micro/go-micro/client"
)

type grpcPublication struct {
	topic       string
	contentType string
	payload     interface{}
}

func newGRPCPublication(topic string, payload interface{}, contentType string, opts ...client.MessageOption) client.Message {
	var options client.MessageOptions
	for _, o := range opts {
		o(&options)
	}

	if len(options.ContentType) > 0 {
		contentType = options.ContentType
	}

	return &grpcPublication{
		payload:     payload,
		topic:       topic,
		contentType: contentType,
	}
}

func (g *grpcPublication) ContentType() string {
	return g.contentType
}

func (g *grpcPublication) Topic() string {
	return g.topic
}

func (g *grpcPublication) Payload() interface{} {
	return g.payload
}
