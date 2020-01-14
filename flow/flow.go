package flow

import (
	"context"
	"time"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/client"
)

type Flow interface {
	// Init flow with options
	Init(...Option) error
	// Get flow options
	Options() Options
	// Create step in flow
	CreateStep(flow string, step *Step) error
	// Remove step from flow
	RemoveStep(flow string, step *Step) error
	// Execute specific flow execution and returns reqID and error
	Execute(ctx context.Context, flow string, req interface{}, rsp interface{}, opts ...ExecuteOption) (string, error)
	// Resume suspended flow execution
	Resume(ctx context.Context, flow string, reqID string) error
	// Pause flow execution
	Pause(ctx context.Context, flow string, reqID string) error
	// Abort flow execution
	Abort(ctx context.Context, flow string, reqID string) error
	// Stop executor and drain active workers
	Stop() error
}

type Step struct {
	// name of step
	Name string
	// operations for step
	Operations []Operation
	// steps that are required for this step
	Requires []string
	// steps for which this step required
	Required []string
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
	StateStore StateStore
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
	// Client to use for communication
	Client client.Client
	// Broker to use for communication
	Broker broker.Broker
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

type ExecutorOption func(*ExecutorOptions)

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

// Client for communication
func ExecuteClient(c client.Client) ExecuteOption {
	return func(o *ExecuteOptions) {
		o.Client = c
	}
}

// Broker for communication
func ExecuteBroker(b broker.Broker) ExecuteOption {
	return func(o *ExecuteOptions) {
		o.Broker = b
	}
}

// State store implementation
func WithStateStore(s Store) Option {
	return func(o *Options) {
		o.StateStore = s
	}
}

// Data store implementation
func WithDataStore(s Store) Option {
	return func(o *Options) {
		o.DataStore = s
	}
}

// Flow store implementation
func WithFlowStore(s Store) ExecutorOption {
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

// Default Timeout for flows
func WithTimeout(td time.Duration) Option {
	return func(o *Options) {
		o.Timeout = td
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
