// Package runtime is a service runtime manager
package runtime

// Runtime is a service runtime manager
type Runtime interface {
	// Registers a service
	Register(*Service) error
	// starts the runtime
	Run() error
	// Shutdown the runtime
	Stop() error
}

type Service struct {
	// name of the service
	Name string
	// url location of source
	Source string
	// path to store source
	Path string
	// exec command
	Exec string
}

var (
	DefaultRuntime = newRuntime()
)

func Register(s *Service) error {
	return DefaultRuntime.Register(s)
}

func Run() error {
	return DefaultRuntime.Run()
}

func Stop() error {
	return DefaultRuntime.Stop()
}
