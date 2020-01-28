package flow

import (
	"log"

	pbFlow "github.com/micro/go-micro/flow/service/proto"
)

func stepToProto(step *Step) *pbFlow.Step {
	ops := make([]*pbFlow.Operation, 0, len(step.Operations))
	for _, op := range step.Operations {
		ops = append(ops, op.Encode())
	}
	fops := make([]*pbFlow.Operation, 0, len(step.Fallback))
	for _, op := range step.Fallback {
		fops = append(fops, op.Encode())
	}

	pb := &pbFlow.Step{
		Name:       step.ID,
		Requires:   step.Requires,
		Required:   step.Required,
		Operations: ops,
		Fallback:   fops,
	}

	return pb
}

func stepEqual(ostep *pbFlow.Step, nstep *pbFlow.Step) bool {
	if ostep.Name == nstep.Name {
		return true
	}

	return false
}

func protoToStep(pb *pbFlow.Step) *Step {
	ops := make([]Operation, 0, len(pb.Operations))
	for _, pbop := range pb.Operations {
		op, ok := Operations[pbop.Type]
		if !ok {
			log.Printf("unknown op %v\n", pbop)
			continue
		}
		nop := op.New()
		nop.Decode(pbop)
		ops = append(ops, nop)
	}

	fops := make([]Operation, 0, len(pb.Fallback))
	for _, pbop := range pb.Fallback {
		op, ok := Operations[pbop.Type]
		if !ok {
			log.Printf("unknown op %v\n", pbop)
			continue
		}
		nop := op.New()
		nop.Decode(pbop)
		fops = append(fops, nop)
	}

	st := &Step{
		ID:         pb.Name,
		Requires:   pb.Requires,
		Required:   pb.Required,
		Fallback:   fops,
		Operations: ops,
	}

	return st
}
