// Package runtime is a service runtime manager
package runtime

import (
	"errors"
	"time"
)

var (
	ErrAlreadyExists = errors.New("already exists")
	ErrNotFound      = errors.New("not found")
)

// Runtime is a service runtime manager
type Runtime interface {
	// Init initializes runtime
	Init(...Option) error
	// CreateNamespace creates a new namespace in the runtime
	CreateNamespace(string) error
	// DeleteNamespace deletes a namespace in the runtime
	DeleteNamespace(string) error
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
	// String defines the runtime implementation
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

// EventType defines schedule event
type EventType int

const (
	// Create is emitted when a new build has been craeted
	Create EventType = iota
	// Update is emitted when a new update become available
	Update
	// Delete is emitted when a build has been deleted
	Delete
)

// String returns human readable event type
func (t EventType) String() string {
	switch t {
	case Create:
		return "create"
	case Delete:
		return "delete"
	case Update:
		return "update"
	default:
		return "unknown"
	}
}

// Event is notification event
type Event struct {
	// ID of the event
	ID string
	// Type is event type
	Type EventType
	// Timestamp is event timestamp
	Timestamp time.Time
	// Service the event relates to
	Service *Service
	// Options to use when processing the event
	Options *CreateOptions
}

// ServiceStatus defines service statuses
type ServiceStatus int

const (
	// Unknown indicates the status of the service is not known
	Unknown ServiceStatus = iota
	// Pending is the initial status of a service
	Pending
	// Building is the status when the service is being built
	Building
	// Starting is the status when the service has been started but is not yet ready to accept traffic
	Starting
	// Running is the status when the service is active and accepting traffic
	Running
	// Stopping is the status when a service is stopping
	Stopping
	// Stopped is the status when a service has been stopped or has completed
	Stopped
	// Error is the status when an error occured, this could be a build error or a run error. The error
	// details can be found within the service's metadata
	Error
)

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
	// Status of the service
	Status ServiceStatus
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
