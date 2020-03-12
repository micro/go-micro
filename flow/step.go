package flow

import (
	pb "github.com/micro/go-micro/v2/flow/service/proto"
)

type Step struct {
	// name of step
	ID string
	// Retry count for step
	Retry int
	// Timeout for step
	Timeout int
	// Step operation to execute
	Operation Operation
	// Which step use as input
	Input string
	// Where to place output
	Output string
	// Steps that are required for this step
	After []string
	// Steps for which this step required
	Before []string
	// Step operation to execute in case of error
	Fallback Operation
}

func (s *Step) Name() string {
	return s.ID
}

func (s *Step) Id() string {
	return s.ID
}

func (s *Step) String() string {
	return s.ID
	//return fmt.Sprintf("step %s, ops: %s, requires: %v, required: %v", s.ID, s.Operations, s.Requires, s.Required)
}

type Steps []*Step

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
		Input:     step.Input,
		Output:    step.Output,
		Operation: step.Operation.Encode(),
	}
	if step.Fallback != nil {
		p.Fallback = step.Fallback.Encode()
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
		Input:     p.Input,
		Output:    p.Output,
		After:     p.After,
		Before:    p.Before,
		Operation: nop,
	}

	if p.Fallback != nil {
		op, ok = Operations[p.Fallback.Type]
		if !ok {
			return nil
		}
		fop := op.New()
		fop.Decode(p.Fallback)
		st.Fallback = fop
	}

	return st
}
