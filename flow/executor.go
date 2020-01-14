package flow

import (
	"context"
	"time"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/client"
)

// Executor run flows and control it exection
type Executor interface {
	// Execuute specific flow execution with data
	Execute(ctx context.Context, flow string, req interface{}, rsp interface{}, opts ...ExecuteOption) (string, error)
	// Resume suspended flow execution
	Resume(ctx context.Context, flow, rid string) error
	// Pause flow execution
	Pause(ctx context.Context, flow, rid string) error
	// Abort flow execution
	Abort(ctx context.Context, flow, rid string) error
	// Init Executor
	Init(...ExecutorOption) error
	// Stop executor and drain active workers
	Stop() error
	// Return options for executor
	Options() ExecutorOptions
}

// Executor options
type ExecutorOptions struct {
	// Number of worker goroutines
	Concurrency int
	// Preallocate worker goroutines
	Prealloc bool
	// If no workers available no blocking
	Nonblock bool
	// Wait completiong before stop
	Wait bool
	// Maximum Timeout for full flow operations
	Timeout time.Duration
	// Client to use for communication
	Client client.Client
	// Broker to use for communication
	Broker broker.Broker
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
func ExecutorWait(b bool) ExecutorOption {
	return func(o *ExecutorOptions) {
		o.Wait = b
	}
}

// Nonnblocking submission
func ExecutorNonblock(b bool) ExecutorOption {
	return func(o *ExecutorOptions) {
		o.Nonblock = b
	}
}

// Panic handler
func ExecutorPanicHandler(h func(interface{})) ExecutorOption {
	return func(o *ExecutorOptions) {
		o.PanicHandler = h
	}
}

// Preallocate goroutine pool
func ExecutorPrealloc(b bool) ExecutorOption {
	return func(o *ExecutorOptions) {
		o.Prealloc = b
	}
}

// Size of goroutine pool
func ExecutorConcurrency(c int) ExecutorOption {
	return func(o *ExecutorOptions) {
		o.Concurrency = c
	}
}

// Client for communication
func ExecutorClient(c client.Client) ExecutorOption {
	return func(o *ExecutorOptions) {
		o.Client = c
	}
}

// Broker for communication
func ExecutorBroker(b broker.Broker) ExecutorOption {
	return func(o *ExecutorOptions) {
		o.Broker = b
	}
}

// State store implementation
func ExecutorStateStore(s StateStore) ExecutorOption {
	return func(o *ExecutorOptions) {
		o.StateStore = s
	}
}

// Data store implementation
func ExecutorDataStore(s DataStore) ExecutorOption {
	return func(o *ExecutorOptions) {
		o.DataStore = s
	}
}

// Flow store implementation
func ExecutorFlowStore(s FlowStore) ExecutorOption {
	return func(o *ExecutorOptions) {
		o.FlowStore = s
	}
}

// Event handler for flow execution
func ExecutorEventHandler(h EventHandler) ExecutorOption {
	return func(o *ExecutorOptions) {
		o.EventHandler = h
	}
}

// Logger
func ExecutorLogger(l Logger) ExecutorOption {
	return func(o *ExecutorOptions) {
		o.Logger = l
	}
}

// Default Timeout for flows
func ExecutorTimeout(td time.Duration) ExecutorOption {
	return func(o *ExecutorOptions) {
		o.Timeout = td
	}
}

// Context store for executor options
func ExecutorContext(ctx context.Context) ExecutorOption {
	return func(o *ExecutorOptions) {
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
