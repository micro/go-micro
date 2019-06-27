// Package health provides health info
package health

const (
	CheckOK Code = iota
	CheckFailed
)

type Code int

type Health interface {
	Register(*Check) error
	Check(id string) (*Status, error)
}

// Check performs some healthcheck
type Check struct {
	// Id of the healthcheck
	Id string
	// The function to execute
	Exec func() error
}

// Status of a healthcheck
type Status struct {
	Code Code
	Info string
}
