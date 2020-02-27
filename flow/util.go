package flow

import (
	pb "github.com/micro/go-micro/v2/flow/service/proto"
)

func stepEqual(oldStep, newStep *Step) bool {
	if oldStep.Name() == newStep.Name() {
		return true
	}
	return false
}

func StepsToProto(steps Steps) *pb.Steps {
	return nil
}

func ProtoToSteps(p *pb.Steps) Steps {
	return nil
}

func StepToProto(step *Step) *pb.Step {
	p := &pb.Step{
		Name:      step.ID,
		After:     step.After,
		Before:    step.Before,
		Operation: step.Operation.Encode(),
	}
	return p
}

func ProtoToStep(p *pb.Step) *Step {
	op, ok := Operations[p.Operation.Type]
	if !ok {
		return nil
	}
	nop := op.New()
	nop.Decode(p.Operation)

	st := &Step{
		ID:        p.Name,
		After:     p.After,
		Before:    p.Before,
		Operation: nop,
	}

	return st
}
