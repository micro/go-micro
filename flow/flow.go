package flow

import (
	"context"
	"fmt"
	"time"
)

type Flow interface {
	// Init flow with options
	Init(...Option) error
	// Get flow options
	Options() Options
	// Create step in specific flow
	CreateStep(ctx context.Context, flow string, step *Step) error
	// Delete step from specific flow
	DeleteStep(ctx context.Context, flow string, step *Step) error
	// Update step in specific flow
	UpdateStep(ctx context.Context, flow string, oldstep *Step, newstep *Step) error
	// Execute specific flow and returns request id and error, optionally fills rsp
	Execute(ctx context.Context, flow string, req interface{}, rsp interface{}, opts ...ExecuteOption) (string, error)
	// Resume specific paused flow execution by request id
	Resume(ctx context.Context, flow string, reqID string) error
	// Pause specific flow execution by request id
	Pause(ctx context.Context, flow string, reqID string) error
	// Abort specific flow execution by request id
	Abort(ctx context.Context, flow string, reqID string) error
	// Stop executor and drain active workers
	Stop() error
}

type Step struct {
	// name of step
	ID string
	// operations for step
	Operations Operations
	// steps that are required for this step
	Requires []string
	// steps for which this step required
	Required []string
}

func (s *Step) Name() string {
	return s.ID
}

func (s *Step) String() string {
	return fmt.Sprintf("step %s, ops: %s, requires: %v, required: %v", s.ID, s.Operations, s.Requires, s.Required)
}

type Option func(*Options)

type Options struct {
	// Number of worker goroutines
	Concurrency int
	// Preallocate worker goroutines
	Prealloc bool
	// If no workers available no blocking
	Nonblock bool
	// Wait completiong before stop
	Wait bool
	// StateStore is used for flow state marking
	StateStore DataStore
	// DataStore is used for intermediate data passed between flow nodes
	DataStore DataStore
	// FlowStore is used for storing flows
	FlowStore FlowStore
	// EventHandler is used to notification about flow progress
	EventHandler EventHandler
	// PanicHandler is used for recovery panics
	PanicHandler func(interface{})
	// Logger is used internally to provide messages
	Logger Logger
	// Context is used for storing non default options
	Context context.Context
}

type ExecuteOptions struct {
	// Timeout for currenct execition
	Timeout time.Duration
	// Async execution run
	Async bool
	// Concurrency specify count of workers create for nodes in flow
	Concurrency int
	// Retries specify count of retries in case of node execution failed
	Retries int
	// Context is used for storing non default options
	Context context.Context
}

type ExecuteOption func(*ExecuteOptions)

// Wait for flow completion before stop
func WithWait(b bool) Option {
	return func(o *Options) {
		o.Wait = b
	}
}

// Nonblocking submission
func WithNonblock(b bool) Option {
	return func(o *Options) {
		o.Nonblock = b
	}
}

// Panic handler
func WithPanicHandler(h func(interface{})) Option {
	return func(o *Options) {
		o.PanicHandler = h
	}
}

// WithPrealloc preallocates goroutine pool
func WithPrealloc(b bool) Option {
	return func(o *Options) {
		o.Prealloc = b
	}
}

// Size of goroutine pool
func WithConcurrency(c int) Option {
	return func(o *Options) {
		o.Concurrency = c
	}
}

// State store implementation
func WithStateStore(s DataStore) Option {
	return func(o *Options) {
		o.StateStore = s
	}
}

// Data store implementation
func WithDataStore(s DataStore) Option {
	return func(o *Options) {
		o.DataStore = s
	}
}

// Flow store implementation
func WithFlowStore(s FlowStore) Option {
	return func(o *Options) {
		o.FlowStore = s
	}
}

// Event handler for flow execution
func WithEventHandler(h EventHandler) Option {
	return func(o *Options) {
		o.EventHandler = h
	}
}

// Logger
func WithLogger(l Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}

// Context store for executor options
func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

// Size of goroutine pool for nodes in flow
func ExecuteConcurrency(c int) ExecuteOption {
	return func(o *ExecuteOptions) {
		o.Concurrency = c
	}
}

// Timeout for specific exection
func ExecuteTimeout(td time.Duration) ExecuteOption {
	return func(o *ExecuteOptions) {
		o.Timeout = td
	}
}

// Number of retries in case of failure
func ExecuteRetries(c int) ExecuteOption {
	return func(o *ExecuteOptions) {
		o.Retries = c
	}
}

// Don't wait for completion
func ExecuteAsync(b bool) ExecuteOption {
	return func(o *ExecuteOptions) {
		o.Async = b
	}
}

// Context for non default options
func ExecuteContext(ctx context.Context) ExecuteOption {
	return func(o *ExecuteOptions) {
		o.Context = ctx
	}
}
