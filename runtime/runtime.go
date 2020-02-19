// Package runtime is a service runtime manager
package runtime

import (
	"errors"
	"time"
)

var (
	// DefaultRuntime is default micro runtime
	DefaultRuntime Runtime = NewRuntime()
	// DefaultName is default runtime service name
	DefaultName = "go.micro.runtime"

	ErrAlreadyExists = errors.New("already exists")
)

// Runtime is a service runtime manager
type Runtime interface {
	// String describes runtime
	String() string
	// Init initializes runtime
	Init(...Option) error
	// Create registers a service
	Create(*Service, ...CreateOption) error
	// Read returns the service
	Read(...ReadOption) ([]*Service, error)
	// Update the service in place
	Update(*Service) error
	// Remove a service
	Delete(*Service) error
	// List the managed services
	List() ([]*Service, error)
	// Start starts the runtime
	Start() error
	// Stop shuts down the runtime
	Stop() error
}

// Scheduler is a runtime service scheduler
type Scheduler interface {
	// Notify publishes schedule events
	Notify() (<-chan Event, error)
	// Close stops the scheduler
	Close() error
}

// EventType defines schedule event
type EventType int

const (
	// Create is emitted when a new deployment has been craeted
	Create EventType = iota
	// Update is emitted when a new update become available
	Update
	// Delete is emitted when a deployment has been deleted
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
	// Type is event type
	Type EventType
	// Timestamp is event timestamp
	Timestamp time.Time
	// Service is the name of the service
	Service string
	// Version of the build
	Version string
}

// StatusType defines the status of a service
type StatusType int

const (
	// Starting means the service has been created
	Starting StatusType = iota
	// Building means the service is building prior to deployment
	Building
	// Deploying means the service is being deployed
	Deploying
	// Running means the service is running and ready
	Running
	// Error means something went wrong
	Error
	// Unknown means the status is unknown
	Unknown
)

func (s StatusType) String() string {
	switch s {
	case Starting:
		return "starting"
	case Building:
		return "building"
	case Deploying:
		return "deploying"
	case Running:
		return "running"
	case Error:
		return "error"
	default:
		return "unknown"
	}
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
	// Status of the service
	Status StatusType
}
