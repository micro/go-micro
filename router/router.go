// Package router provides a network routing control plane
package router

import (
	"time"
)

// Router is an interface for a routing control plane
type Router interface {
	// Init initializes the router with options
	Init(...Option) error
	// Options returns the router options
	Options() Options
	// The routing table
	Table() Table
	// Advertise advertises routes to the network
	Advertise() (<-chan *Advert, error)
	// Process processes incoming adverts
	Process(*Advert) error
	// Lookup queries routes in the routing table
	Lookup(Query) ([]Route, error)
	// Watch returns a watcher which tracks updates to the routing table
	Watch(opts ...WatchOption) (Watcher, error)
	// Start starts the router
	Start() error
	// Status returns router status
	Status() Status
	// Stop stops the router
	Stop() error
	// Returns the router implementation
	String() string
}

type Table interface {
	// Create new route in the routing table
	Create(Route) error
	// Delete deletes existing route from the routing table
	Delete(Route) error
	// Update updates route in the routing table
	Update(Route) error
	// List returns the list of all routes in the table
	List() ([]Route, error)
	// Query queries routes in the routing table
	Query(Query) ([]Route, error)
}

// Option used by the router
type Option func(*Options)

// StatusCode defines router status
type StatusCode int

const (
	// Running means the router is up and running
	Running StatusCode = iota
	// Advertising means the router is advertising
	Advertising
	// Stopped means the router has been stopped
	Stopped
	// Error means the router has encountered error
	Error
)

func (s StatusCode) String() string {
	switch s {
	case Running:
		return "running"
	case Advertising:
		return "advertising"
	case Stopped:
		return "stopped"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

// Status is router status
type Status struct {
	// Code defines router status
	Code StatusCode
	// Error contains error description
	Error error
}

// String returns human readable status
func (s Status) String() string {
	return s.Code.String()
}

// AdvertType is route advertisement type
type AdvertType int

const (
	// Announce is advertised when the router announces itself
	Announce AdvertType = iota
	// RouteUpdate advertises route updates
	RouteUpdate
)

// String returns human readable advertisement type
func (t AdvertType) String() string {
	switch t {
	case Announce:
		return "announce"
	case RouteUpdate:
		return "update"
	default:
		return "unknown"
	}
}

// Advert contains a list of events advertised by the router to the network
type Advert struct {
	// Id is the router Id
	Id string
	// Type is type of advert
	Type AdvertType
	// Timestamp marks the time when the update is sent
	Timestamp time.Time
	// TTL is Advert TTL
	TTL time.Duration
	// Events is a list of routing table events to advertise
	Events []*Event
}

var (
	// DefaultAddress is default router address
	DefaultAddress = ":9093"
	// DefaultName is default router service name
	DefaultName = "go.micro.router"
	// DefaultNetwork is default micro network
	DefaultNetwork = "go.micro"
	// DefaultRouter is default network router
	DefaultRouter = NewRouter()
)

// NewRouter creates new Router and returns it
func NewRouter(opts ...Option) Router {
	return newRouter(opts...)
}
