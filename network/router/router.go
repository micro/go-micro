// Package router provides a network routing control plane
package router

import (
	"time"

	"github.com/micro/go-micro/network/router/table"
)

const (
	// Status codes
	// Running means the router is up and running
	Running StatusCode = iota
	// Stopped means the router has been stopped
	Stopped
	// Error means the router has encountered error
	Error

	// Advert types
	// Announce is advertised when the router announces itself
	Announce AdvertType = iota
	// Update advertises route updates
	Update
)

// Router is an interface for a routing control plane
type Router interface {
	// Router provides a routing table
	table.Table
	// Init initializes the router with options
	Init(...Option) error
	// Options returns the router options
	Options() Options
	// Advertise advertises routes to the network
	Advertise() (<-chan *Advert, error)
	// Process processes incoming adverts
	Process(*Advert) error
	// Status returns router status
	Status() Status
	// Stop stops the router
	Stop() error
	// Returns the router implementation
	String() string
}

// Option used by the router
type Option func(*Options)

// AdvertType is route advertisement type
type AdvertType int

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

var (
	// DefaultRouter is default network router
	DefaultRouter = NewRouter()
)

// NewRouter creates new Router and returns it
func NewRouter(opts ...Option) Router {
	return newRouter(opts...)
}
