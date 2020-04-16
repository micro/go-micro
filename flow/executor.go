package flow

type Executor interface {
	// Init flow with options
	Init(...ExecutorOption) error
	// Get flow options
	Options() ExecutorOptions
	// Run execution
	Execute(steps []*Step, req interface{}, rsp interface{}, opts ...ExecuteOption) (string, error)
	// Resume specific paused flow execution by request id
	Resume(flow string, id string) error
	// Pause specific flow execution by request id
	Pause(flow string, id string) error
	// Abort specific flow execution by request id
	Abort(flow string, id string) error
	// Status show status specific flow execution by request id
	Status(flow string, id string) (Status, error)
	// Result get result of the flow step
	Result(flow string, id string, step string) ([]byte, error)
	// Stop executor and drain active workers
	Stop() error
}
