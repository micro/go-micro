package flow

import (
	"context"

	pbFlow "github.com/micro/go-micro/v2/flow/service/proto"
)

type emptyOperation struct {
	name    string
	options OperationOptions
}

func EmptyOperation(opts ...OperationOption) *emptyOperation {
	options := OperationOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return &emptyOperation{name: "empty_operation", options: options}
}

func (op *emptyOperation) New() Operation {
	return &emptyOperation{}
}

func (op *emptyOperation) Execute(ctx context.Context, req []byte, opts ...ExecuteOption) ([]byte, error) {
	return nil, nil
}

func (op *emptyOperation) Name() string {
	return op.name
}

func (op *emptyOperation) Type() string {
	return "empty_operation"
}

func (op *emptyOperation) String() string {
	return op.name
}

func (op *emptyOperation) Encode() *pbFlow.Operation {
	return nil
}

func (op *emptyOperation) Decode(pb *pbFlow.Operation) {
}

func (op *emptyOperation) Options() OperationOptions {
	return op.options
}
