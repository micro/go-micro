package flow

import (
	"context"

	//"github.com/micro/go-micro/v2/broker"
	"github.com/micro/go-micro/v2/codec/bytes"
	pbFlow "github.com/micro/go-micro/v2/flow/service/proto"
)

type clientPublishOperation struct {
	name    string
	topic   string
	options OperationOptions
}

func ClientPublishOperation(topic string, opts ...OperationOption) *clientPublishOperation {
	options := OperationOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return &clientPublishOperation{
		name:    topic,
		topic:   topic,
		options: options,
	}
}

func (op *clientPublishOperation) New() Operation {
	return &clientPublishOperation{}
}

func (op *clientPublishOperation) Execute(ctx context.Context, data []byte, opts ...ExecuteOption) ([]byte, error) {
	var err error

	options := ExecuteOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	err = options.Client.Publish(ctx, options.Client.NewMessage(op.topic, &bytes.Frame{Data: data}))
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (op *clientPublishOperation) Name() string {
	return op.name
}

func (op *clientPublishOperation) Type() string {
	return "client_publish_operation"
}

func (op *clientPublishOperation) String() string {
	return op.name
}

func (op *clientPublishOperation) Encode() *pbFlow.Operation {
	pb := &pbFlow.Operation{
		Name:    op.name,
		Type:    op.Type(),
		Options: make(map[string]string),
	}
	pb.Options["topic"] = op.topic
	return pb
}

func (op *clientPublishOperation) Decode(pb *pbFlow.Operation) {
	op.name = pb.Name
	op.topic = pb.Options["topic"]
}

func (op *clientPublishOperation) Options() OperationOptions {
	return op.options
}
