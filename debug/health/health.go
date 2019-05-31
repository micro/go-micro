// Package health is for user defined health checks
package health

type Health interface {
	Register([]*Check) error
	Read(...ReadOption) ([]*Check, error)
	Check(...CheckOption) ([]*Status, error)
	String() string
}

type Check struct {
	Id       string
	Metadata map[string]string
	Exec     func() (*Status, error)
}

type Status struct {
	Code   int
	Detail string
}

type CheckOptions struct{}

type CheckOption func(o *CheckOptions)

type ReadOptions struct{}

type ReadOption func(o *ReadOptions)
