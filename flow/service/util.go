package service

import (
	"log"

	"github.com/micro/go-micro/v2/flow"
	pbFlow "github.com/micro/go-micro/v2/flow/service/proto"
)

func stepToProto(step *flow.Step) *pbFlow.Step {
	operations := make([]*pbFlow.Operation, 0, len(step.Operations))
	for _, op := range step.Operations {
		operations = append(operations, op.Encode())
	}

	pb := &pbFlow.Step{
		Name:       step.ID,
		Requires:   step.Requires,
		Required:   step.Required,
		Operations: operations,
	}

	return pb
}

func protoToStep(pb *pbFlow.Step) *flow.Step {
	ops := make([]flow.Operation, 0, len(pb.Operations))
	for _, pbop := range pb.Operations {
		op, ok := flow.Operations[pbop.Type]
		if !ok {
			log.Printf("unknown op %v\n", pbop)
			continue
		}
		nop := op.New()
		nop.Decode(pbop)
		ops = append(ops, nop)
	}

	st := &flow.Step{
		ID:         pb.Name,
		Requires:   pb.Requires,
		Required:   pb.Required,
		Operations: ops,
	}

	return st
}
