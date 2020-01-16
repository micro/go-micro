package flow

import (
	"log"

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
	ops := make([]Operation, 0, len(pb.Operations))
	for _, pbop := range pb.Operations {
		op, ok := operations[pbop.Type]
		if !ok {
			log.Printf("unknown op %v\n", pbop)
			continue
		}
		op.Decode(pbop)
		ops = append(ops, op)
	}

	st := &Step{
		Name:       pb.Name,
		Requires:   pb.Requires,
		Required:   pb.Required,
		Operations: ops,
	}

	return st
}
