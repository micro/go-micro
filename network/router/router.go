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
	Advertise() (<-chan *Advert, error)
	// Update updates the routing table
	Update(*Advert) error
	// Status returns router status
	Status() Status
	// Stop stops the router
	Stop() error
	// String returns debug info
	String() string
}

// Option used by the router
type Option func(*Options)

// UpdateType is route advertisement update type
type UpdateType int

const (
	// Announce is advertised when the router announces itself
	Announce UpdateType = iota
	// Update advertises route updates
	Update
)

// String returns string representation of update event
func (ut UpdateType) String() string {
	switch ut {
	case Announce:
		return "ANNOUNCE"
	case Update:
		return "UPDATE"
	default:
		return "UNKNOWN"
	}
}

// Advert is sent by the router to the network
type Advert struct {
	// ID is the router ID
	ID string
	// Timestamp marks the time when the update is sent
	Timestamp time.Time
	// Events is a list of events to advertise
	Events []*Event
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
	// Running means the router is up and running
	Running
	// Stopped means the router has been stopped
	Stopped
	// Error means the router has encountered error
	Error
)

// String returns human readable status code
func (sc StatusCode) String() string {
	switch sc {
	case Init:
		return "INITIALIZED"
	case Running:
		return "RUNNING"
	case Stopped:
		return "STOPPED"
	case Error:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// NewRouter creates new Router and returns it
func NewRouter(opts ...Option) Router {
	return newRouter(opts...)
}
