package flow

import (
	"context"
	"fmt"

	pb "github.com/micro/go-micro/v2/flow/service/proto"
)

type emptyOperation struct {
	name    string
	options OperationOptions
}

func EmptyOperation(name string, opts ...OperationOption) *emptyOperation {
	options := OperationOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return &emptyOperation{name: name, options: options}
}

func (op *emptyOperation) New() Operation {
	return &emptyOperation{}
}

func (op *emptyOperation) Execute(ctx context.Context, req []byte, opts ...ExecuteOption) ([]byte, error) {
	fmt.Printf("execute %s\n", op.name)
	return req, nil
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

func (op *emptyOperation) Encode() *pb.Operation {
	return &pb.Operation{Name: op.Name(), Type: op.Type()}
}

func (op *emptyOperation) Decode(p *pb.Operation) {
	op.name = p.Name
}

func (op *emptyOperation) Options() OperationOptions {
	return op.options
}
