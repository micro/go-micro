package flow

type Executor interface {
	// Init flow with options
	Init(...Option) error
	// Get flow options
	Options() Options
	Run(flow string, req interface{}, rsp interface{}, opts ...ExecuteOption) (string, error)
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
