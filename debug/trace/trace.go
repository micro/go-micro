// Package trace provides a tracing interface
package trace

import (
	"time"
)

// Trace is for request tracing
type Trace interface {
	// Read the traces
	Read(...ReadOption) ([]*Span, error)
	// Collect traces
	Collect([]*Span) error
	// Name of tracer
	String() string
}

type Span struct {
	Id       string
	Name     string
	Trace    string
	Metadata map[string]string
	Start    time.Time
	Finish   time.Time
}

type ReadOptions struct{}

type ReadOption func(o *ReadOptions)
