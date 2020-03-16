package flow

import (
	"fmt"

	"github.com/micro/go-micro/v2/store/memory"
)

var (
	ErrStepExists = fmt.Errorf("step already exists")
	DefaultFlow   = newMicroFlow()
	DefaultStore  = memory.NewStore()
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
}
