package flow

import (
	"log"

	pbFlow "github.com/micro/go-micro/flow/service/proto"
)

func stepToProto(step *Step) *pbFlow.Step {
	pb := &pbFlow.Step{
		Name:      step.ID,
		Requires:  step.Requires,
		Required:  step.Required,
		Operation: step.Operation.Encode(),
		Fallback:  step.Fallback.Encode(),
	}

	return pb
}

func stepEqual(ostep *Step, nstep *Step) bool {
	if ostep.Name() == nstep.Name() {
		return true
	}

	return false
}

func protoToStep(pb *pbFlow.Step) *Step {
	var nop Operation
	var fop Operation

	if pb.Operation != nil {
		op, ok := Operations[pb.Operation.Type]
		if !ok {
			log.Printf("unknown op %v\n", pb.Operation)
			return nil
		}
		nop = op.New()
		nop.Decode(pb.Operation)
	}

	if pb.Fallback != nil {
		op, ok := Operations[pb.Fallback.Type]
		if !ok {
			log.Printf("unknown op %v\n", pb.Fallback)
			return nil
		}
		fop = op.New()
		fop.Decode(pb.Fallback)
	}

	st := &Step{
		ID:        pb.Name,
		Requires:  pb.Requires,
		Required:  pb.Required,
		Fallback:  fop,
		Operation: nop,
	}

	return st
}
