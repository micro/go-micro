// Package status provides the ability to set status events
package status

import (
	"time"
)

const (
	Unknown Code = iota
	Started
	Running
	Stopped
	Failed
	OK
)

type Code int

// Status is for setting status
type Status interface {
	// Get the status
	Get(service string) (*Event, error)
	// History returns the service status history
	History(service string) ([]*Event, error)
	// Updates the status
	Update(service string, ev *Event) error
	// Returns status updates
	Notify() (<-chan *Event, error)
}

// A status event
type Event struct {
	// The service for this event
	Service string
	// The unique id of this event
	Id string
	// The time of the event
	Timestamp time.Time
	// Informational message
	Message string
	// The status code
	Type Code
}
