// Package router provides a network routing control plane
package router

import "time"

var (
	// DefaultRouter is default network router
	DefaultRouter = NewRouter()
)

// Router is an interface for a routing control plane
type Router interface {
	// Init initializes the router with options
	Init(...Option) error
	// Options returns the router options
	Options() Options
	// ID returns the id of the router
	ID() string
	// Table returns the routing table
	Table() Table
	// Address returns the router adddress
	Address() string
	// Network returns the network address of the router
	Network() string
	// Advertise starts advertising routes to the network
	Advertise() (<-chan *Update, error)
	// Update updates the routing table
	Update(*Update) error
	// Status returns router status
	Status() Status
	// Stop stops the router
	Stop() error
	// String returns debug info
	String() string
}

// Update is sent by the router to the network
type Update struct {
	// ID is the router ID
	ID string
	// Timestamp marks the time when update is sent
	Timestamp time.Time
	// Event defines advertisement even
	Event *Event
}

// StatusCode defines router status
type StatusCode int

// Status is router status
type Status struct {
	// Error is router error
	Error error
	// Code defines router status
	Code StatusCode
}

const (
	// Init means the rotuer has just been initialized
	Init StatusCode = iota
	// Running means the router is running
	Running
	// Error means the router has crashed with error
	Error
	// Stopped means the router has stopped
	Stopped
)

// String returns human readable status code
func (sc StatusCode) String() string {
	switch sc {
	case Init:
		return "INITIALIZED"
	case Running:
		return "RUNNING"
	case Error:
		return "ERROR"
	case Stopped:
		return "STOPPED"
	default:
		return "UNKNOWN"
	}
}

// Option used by the router
type Option func(*Options)

// NewRouter creates new Router and returns it
func NewRouter(opts ...Option) Router {
	return newRouter(opts...)
}
