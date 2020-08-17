// Package runtime is a service runtime manager
package runtime

import (
	"errors"

	"github.com/micro/go-micro/v3/events"
)

var (
	ErrAlreadyExists = errors.New("already exists")
)

// Runtime is a service runtime manager
type Runtime interface {
	// Init initializes runtime
	Init(...Option) error
	// Create registers a service
	Create(*Service, ...CreateOption) error
	// Read returns the service
	Read(...ReadOption) ([]*Service, error)
	// Update the service in place
	Update(*Service, ...UpdateOption) error
	// Remove a service
	Delete(*Service, ...DeleteOption) error
	// Logs returns the logs for a service
	Logs(*Service, ...LogsOption) (Logs, error)
	// Start starts the runtime
	Start() error
	// Stop shuts down the runtime
	Stop() error
	// String describes runtime
	String() string
}

// Logs returns a log stream
type Logs interface {
	Error() error
	Chan() chan Log
	Stop() error
}

// Log is a log message
type Log struct {
	Message  string
	Metadata map[string]string
}

// Scheduler is a runtime service scheduler
type Scheduler interface {
	// Notify publishes schedule events
	Notify() (<-chan events.Event, error)
	// Close stops the scheduler
	Close() error
}

// Service is runtime service
type Service struct {
	// Name of the service
	Name string
	// Version of the service
	Version string
	// url location of source
	Source string
	// Metadata stores metadata
	Metadata map[string]string
}

// Resources which are allocated to a serivce
type Resources struct {
	// CPU is the maximum amount of CPU the service will be allocated (unit millicpu)
	// e.g. 0.25CPU would be passed as 250
	CPU int
	// Mem is the maximum amount of memory the service will be allocated (unit mebibyte)
	// e.g. 128 MiB of memory would be passed as 128
	Mem int
	// Disk is the maximum amount of disk space the service will be allocated (unit mebibyte)
	// e.g. 128 MiB of memory would be passed as 128
	Disk int
}
