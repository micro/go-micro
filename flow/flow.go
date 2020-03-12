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
