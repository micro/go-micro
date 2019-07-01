// Package health provides health info
package health

import (
	"time"
)

const (
	CheckOK Code = iota
	CheckFailed
)

type Code int

// Health manages the health of services
type Health interface {
	// Register a healthcheck
	Register(id string, c *Check) error
	// Check the health
	Check(id string) (*Result, error)
	// Get healthcheck history
	History(id string) ([]*Result, error)
	// Notify of check results
	Notify() (<-chan *Result, error)
}

// Check is a healthcheck function
type Check func() error

// Result is the result of a healthcheck
type Result struct {
	// check id
	Id string
	// time of status check
	Timestamp time.Time
	// the status code
	Code Code
	// info message
	Info string
}
