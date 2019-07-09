// Package router provides a network routing control plane
package router

import (
	"time"

	"github.com/micro/go-micro/network/router/table"
)

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
	// ID returns the ID of the router
	ID() string
	// Address returns the router adddress
	Address() string
	// Network returns the network address of the router
	Network() string
	// Table returns the routing table
	Table() table.Table
	// Advertise advertises routes to the network
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

// AdvertType is route advertisement type
type AdvertType int

const (
	// Announce is advertised when the router announces itself
	Announce AdvertType = iota
	// Update advertises route updates
	Update
)

// String returns string representation of update event
func (at AdvertType) String() string {
	switch at {
	case Announce:
		return "ANNOUNCE"
	case Update:
		return "UPDATE"
	default:
		return "UNKNOWN"
	}
}

// Advert contains a list of events advertised by the router to the network
type Advert struct {
	// ID is the router ID
	ID string
	// Type is type of advert
	Type AdvertType
	// Timestamp marks the time when the update is sent
	Timestamp time.Time
	// TTL is Advert TTL
	// TODO: not used
	TTL time.Time
	// Events is a list of routing table events to advertise
	Events []*table.Event
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
	// Running means the router is up and running
	Running StatusCode = iota
	// Stopped means the router has been stopped
	Stopped
	// Error means the router has encountered error
	Error
)

// String returns human readable status code
func (sc StatusCode) String() string {
	switch sc {
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
