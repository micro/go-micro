package flow

import (
	"context"
	"fmt"

	pbFlow "github.com/micro/go-micro/v2/flow/service/proto"
)

type flowExecuteOperation struct {
	name      string
	flow      string
	operation string
	options   OperationOptions
}

func FlowExecuteOperation(flow string, op string) *flowExecuteOperation {
	return &flowExecuteOperation{name: fmt.Sprintf("%s.%s", flow, op), flow: flow, operation: op}
}

func (op *flowExecuteOperation) New() Operation {
	return &flowExecuteOperation{}
}

func (op *flowExecuteOperation) Name() string {
	return op.name
}

func (op *flowExecuteOperation) String() string {
	return op.name
}

func (op *flowExecuteOperation) Type() string {
	return "flow_execute_operation"
}

func (op *flowExecuteOperation) Encode() *pbFlow.Operation {
	pb := &pbFlow.Operation{
		Name:    op.name,
		Type:    op.Type(),
		Options: make(map[string]string),
	}
	pb.Options["flow"] = op.flow
	pb.Options["op"] = op.operation
	return pb
}

func (op *flowExecuteOperation) Decode(pb *pbFlow.Operation) {
	op.name = pb.Name
	op.flow = pb.Options["flow"]
	op.operation = pb.Options["op"]
}

func (op *flowExecuteOperation) Options() OperationOptions {
	return op.options
}

func (op *flowExecuteOperation) Execute(ctx context.Context, req []byte, opts ...ExecuteOption) ([]byte, error) {
	fl, err := FlowFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var rsp []byte
	opts = append(opts, ExecuteStep(op.operation), ExecuteFlow(op.flow))
	_, err = fl.Execute(req, rsp, opts...)
	return rsp, err
}
