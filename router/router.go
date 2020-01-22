// Package router provides a network routing control plane
package router

import (
	"time"
)

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

// Router is an interface for a routing control plane
type Router interface {
	// Init initializes the router with options
	Init(...Option) error
	// Options returns the router options
	Options() Options
	// The routing table
	Table() Table
	// Advertise advertises routes
	Advertise() (<-chan *Advert, error)
	// Process processes incoming adverts
	Process(*Advert) error
	// Lookup queries routes in the routing table
	Lookup(...QueryOption) ([]Route, error)
	// Watch returns a watcher which tracks updates to the routing table
	Watch(opts ...WatchOption) (Watcher, error)
	// Start starts the router
	Start() error
	// Stop stops the router
	Stop() error
	// Returns the router implementation
	String() string
}

// Table is an interface for routing table
type Table interface {
	// Create new route in the routing table
	Create(Route) error
	// Delete existing route from the routing table
	Delete(Route) error
	// Update route in the routing table
	Update(Route) error
	// List all routes in the table
	List() ([]Route, error)
	// Query routes in the routing table
	Query(...QueryOption) ([]Route, error)
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

// Strategy is route advertisement strategy
type Strategy int

// TODO: remove the "Advertise" prefix from these
const (
	// AdvertiseAll advertises all routes to the network
	AdvertiseAll Strategy = iota
	// AdvertiseBest advertises optimal routes to the network
	AdvertiseBest
	// AdvertiseLocal will only advertise the local routes
	AdvertiseLocal
	// AdvertiseNone will not advertise any routes
	AdvertiseNone
)

// String returns human readable Strategy
func (s Strategy) String() string {
	switch s {
	case AdvertiseAll:
		return "all"
	case AdvertiseBest:
		return "best"
	case AdvertiseLocal:
		return "local"
	case AdvertiseNone:
		return "none"
	default:
		return "unknown"
	}
}

// NewRouter creates new Router and returns it
func NewRouter(opts ...Option) Router {
	return newRouter(opts...)
}
