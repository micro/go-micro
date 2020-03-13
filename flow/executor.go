package flow

type Executor interface {
	// Init flow with options
	Init(...ExecutorOption) error
	// Get flow options
	Options() Options
	// Run execution
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
	Result(flow string, reqID string, step string) ([]byte, error)
	// Stop executor and drain active workers
	Stop() error
}
