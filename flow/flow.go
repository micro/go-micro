package flow

import (
	"fmt"

	"github.com/micro/go-micro/v2/store/memory"
)

var (
	ErrStepExists   = fmt.Errorf("step already exists")
	DefaultFlow     = newMicroFlow()
	DefaultExecutor = newMicroExecutor()
	DefaultStore    = memory.NewStore()
)

type Flow interface {
	// Init flow with options
	Init(...Option) error
	// Get flow options
	Options() Options
	// Create step in specific flow
	CreateStep(flow string, step *Step) error
	// Delete step from specific flow
	DeleteStep(flow string, step *Step) error
	// Replace step in specific flow
	ReplaceStep(flow string, oldstep *Step, newstep *Step) error
	// Lookup specific flow
	Lookup(flow string) ([]*Step, error)
	// Execute specific flow and returns request id and error, optionally fills rsp
	Execute(flow string, req interface{}, rsp interface{}, opts ...ExecuteOption) (string, error)
	// Resume specific paused flow execution by request id
	Resume(flow string, reqID string) error
	// Pause specific flow execution by request id
	Pause(flow string, reqID string) error
	// Abort specific flow execution by request id
	Abort(flow string, reqID string) error
	// Status show status specific flow execution by request id
	Status(flow string, reqID string) (Status, error)
	// Result get result of the flow step
	Result(flow string, reqID string, step *Step) ([]byte, error)
	// Stop executor and drain active workers
	Stop() error
}

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
