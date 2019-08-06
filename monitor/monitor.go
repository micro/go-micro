// Package monitor monitors service health
package monitor

import (
	"errors"
)

const (
	StatusUnknown StatusCode = iota
	StatusRunning
	StatusFailed
)

type StatusCode int

// Monitor monitors a service and reaps dead instances
type Monitor interface {
	// Status of the service
	Status(service string) (Status, error)
	// Watch starts watching the service
	Watch(service string) error
	// Stop monitoring
	Stop() error
}

type Status struct {
	Code  StatusCode
	Info  string
	Error string
}

var (
	ErrNotWatching = errors.New("not watching")
)

// NewMonitor returns a new monitor
func NewMonitor(opts ...Option) Monitor {
	return newMonitor(opts...)
}
