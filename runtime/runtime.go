// Package runtime is a service runtime manager
package runtime

import (
	"time"

	"github.com/micro/go-micro/runtime/build"
)

var (
	// DefaultRuntime is default micro runtime
	DefaultRuntime = newRuntime()
)

// Runtime is a service runtime manager
type Runtime interface {
	// Registers a service
	Create(*Service, ...CreateOption) error
	// Remove a service
	Delete(*Service) error
	// Update the service in place
	Update(*Service) error
	// List the managed services
	List() ([]*Service, error)
	// starts the runtime
	Start() error
	// Shutdown the runtime
	Stop() error
}

// Poller periodically poll for updates and returns the results
type Poller interface {
	// Poll polls for updates and returns results
	Poll() (*build.Build, error)
	// Tick returns poller tick time
	Tick() time.Duration
}

// Service is runtime service
type Service struct {
	// Name of the service
	Name string
	// url location of source
	Source string
	// Path to store source
	Path string
	// Exec command
	Exec string
	// Version of the service
	Version string
}

func Create(s *Service, opts ...CreateOption) error {
	return DefaultRuntime.Create(s, opts...)
}

func Delete(s *Service) error {
	return DefaultRuntime.Delete(s)
}

func Update(s *Service) error {
	return DefaultRuntime.Update(s)
}

func List() ([]*Service, error) {
	return DefaultRuntime.List()
}

func Start() error {
	return DefaultRuntime.Start()
}

func Stop() error {
	return DefaultRuntime.Stop()
}
