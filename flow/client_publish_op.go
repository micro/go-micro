package flow

import (
	"context"
	"fmt"

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

func (op *clientPublishOperation) Clone() Operation {
	return &clientPublishOperation{}
}

func (op *clientPublishOperation) Execute(ctx context.Context, data []byte, opts ...ExecuteOption) ([]byte, error) {
	var err error
	var rsp []byte

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
	md["Micro-Callback"] = fmt.Sprintf("%s-%s", op.topic, options.ID)
	md["Micro-Flow"] = options.Flow

	done := make(chan struct{}, 1)
	sub, err := options.Client.Options().Broker.Subscribe(md["Micro-Callback"], func(evt broker.Event) error {
		rsp = make([]byte, len(evt.Message().Body))
		copy(rsp, evt.Message().Body)
		err := evt.Ack()
		done <- struct{}{}
		//close(done)
		return err
	}, broker.SubscribeContext(ctx))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = sub.Unsubscribe()
	}()

	err = options.Client.Options().Broker.Publish(op.topic, &broker.Message{Header: md, Body: data})
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout")
	case <-done:
		break
	}

	return rsp, nil
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
