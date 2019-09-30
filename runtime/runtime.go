// Package runtime is a service runtime manager
package runtime

// Runtime is a service runtime manager
type Runtime interface {
	// Registers a service
	Create(*Service, ...CreateOption) error
	// Remove a service
	Delete(*Service) error
	// starts the runtime
	Start() error
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

func Create(s *Service, opts ...CreateOption) error {
	return DefaultRuntime.Create(s, opts...)
}

func Delete(s *Service) error {
	return DefaultRuntime.Delete(s)
}

func Start() error {
	return DefaultRuntime.Start()
}

func Stop() error {
	return DefaultRuntime.Stop()
}
