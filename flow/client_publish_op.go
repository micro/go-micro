package flow

import (
	"context"
	"fmt"
	"log"

	"github.com/micro/go-micro/v2/broker"
	pbFlow "github.com/micro/go-micro/v2/flow/service/proto"
	"github.com/micro/go-micro/v2/metadata"
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

	fl, err := FlowFromContext(ctx)
	if err != nil {
		return nil, err
	}
	_ = fl
	//_, err = fl.Execute(op.flow, req, nil, opts...)

	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = make(map[string]string)
	}

	// standard micro headers
	md["Content-Type"] = options.Client.Options().ContentType
	md["Micro-Topic"] = op.topic
	md["Micro-Id"] = options.ID
	// header to send reply back
	md["Micro-Response"] = fmt.Sprintf("%s-%s", op.topic, options.ID)

	sub, err := options.Broker.Subscribe(md["Micro-Response"], func(evt broker.Event) error {
		log.Printf("RSP %#+v\n")
		return evt.Ack()
	}, broker.SubscribeContext(ctx))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = sub.Unsubscribe()
	}()

	err = options.Broker.Publish(op.topic, &broker.Message{Header: md, Body: data})
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
