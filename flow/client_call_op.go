package flow

import (
	"context"
	"fmt"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/codec/bytes"
	pbFlow "github.com/micro/go-micro/v2/flow/service/proto"
)

type clientCallOperation struct {
	name     string
	service  string
	endpoint string
	options  OperationOptions
}

func ClientCallOperation(service, endpoint string, opts ...OperationOption) *clientCallOperation {
	options := OperationOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return &clientCallOperation{
		name:     fmt.Sprintf("%s.%s", service, endpoint),
		service:  service,
		endpoint: endpoint,
		options:  options,
	}
}

func (op *clientCallOperation) New() Operation {
	return &clientCallOperation{}
}

func (op *clientCallOperation) Execute(ctx context.Context, data []byte, opts ...ExecuteOption) ([]byte, error) {
	var err error

	options := ExecuteOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	req := client.NewRequest(op.service, op.endpoint, &bytes.Frame{Data: data})
	rsp := &bytes.Frame{}

	copts := []client.CallOption{}
	if opts, ok := options.Context.Value(clientCallOperation{}).([]client.CallOption); ok {
		copts = opts
	}

	if err = options.Client.Call(ctx, req, rsp, copts...); err != nil {
		fmt.Printf("%s.%s: %v\n", op.service, op.endpoint, err)
		return nil, err
	}

	return rsp.Data, nil
}

func (op *clientCallOperation) Name() string {
	return op.name
}

func (op *clientCallOperation) Type() string {
	return "client_call_operation"
}

func (op *clientCallOperation) String() string {
	return op.name
}

func (op *clientCallOperation) Encode() *pbFlow.Operation {
	pb := &pbFlow.Operation{
		Name:    op.name,
		Type:    op.Type(),
		Options: make(map[string]string),
	}
	pb.Options["service"] = op.service
	pb.Options["endpoint"] = op.endpoint
	return pb
}

func (op *clientCallOperation) Decode(pb *pbFlow.Operation) {
	op.name = pb.Name
	op.service = pb.Options["service"]
	op.endpoint = pb.Options["endpoint"]
}

func (op *clientCallOperation) Options() OperationOptions {
	return op.options
}
