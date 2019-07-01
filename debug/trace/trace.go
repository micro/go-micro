// Package trace is for stack tracing
package trace

import (
	"time"
)

// Trace provides stack tracing recording
type Trace interface {
	// return a stack trace
	Get(id string) (*Stack, error)
	// Record against a stack
	Record(id string, f *Frame) error
}

// Stack is a stack trace
type Stack struct {
	// Id of the stack
	Id string
	// Frames for the stack trace
	Frames []*Frame
}

// Frame is a frame in the stack trace
type Frame struct {
	// Id of this frame
	Id string
	// Time of the recording
	Timestamp time.Time
	// the frame context
	Context map[string]interface{}
	// the frame data
	Data []byte
}
