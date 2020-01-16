package flow

import (
	pbFlow "github.com/micro/go-micro/flow/proto"
)

func stepToProto(step *Step) *pbFlow.Step {
	operations := make([]*pbFlow.Operation, 0, len(step.Operations))
	for _, op := range step.Operations {
		operations = append(operations, op.Encode())
	}

	pb := &pbFlow.Step{
		Name:       step.Name,
		Requires:   step.Requires,
		Required:   step.Required,
		Operations: operations,
	}

	return pb
}

func protoToStep(pb *pbFlow.Step) *Step {
	return nil
}
