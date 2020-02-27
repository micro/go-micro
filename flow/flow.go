package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/micro/go-micro/v2/broker"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/store"
)

type Flow interface {
	// Init flow with options
	Init(...Option) error
	// Get flow options
	Options() Options
	// Create step in specific flow
	CreateStep(flow string, step *Step) error
	// Delete step from specific flow
	DeleteStep(flow string, step *Step) error
	// Replace step in specific flow
	ReplaceStep(flow string, oldstep *Step, newstep *Step) error
	// Lookup specific flow
	Lookup(flow string) ([]*Step, error)
	// Execute specific flow and returns request id and error, optionally fills rsp
	Execute(flow string, req interface{}, rsp interface{}, opts ...ExecuteOption) (string, error)
	// Resume specific paused flow execution by request id
	Resume(flow string, reqID string) error
	// Pause specific flow execution by request id
	Pause(flow string, reqID string) error
	// Abort specific flow execution by request id
	Abort(flow string, reqID string) error
	// Status show status specific flow execution by request id
	Status(flow string, reqID string) (Status, error)
	// Result get result of the flow step
	Result(flow string, reqID string, step *Step) ([]byte, error)
	// Stop executor and drain active workers
	Stop() error
}

type Step struct {
	// name of step
	ID string
	// Retry count for step
	Retry int
	// Timeout for step
	Timeout int
	// Step operation to execute
	Operation Operation
	// Which step use as input
	Input string
	// Where to place output
	Output string
	// Steps that are required for this step
	After []string
	// Steps for which this step required
	Before []string
	// Step operation to execute in case of error
	Fallback Operation
}

func (s *Step) Name() string {
	return s.ID
}

func (s *Step) Id() string {
	return s.ID
}

func (s *Step) String() string {
	return s.ID
	//return fmt.Sprintf("step %s, ops: %s, requires: %v, required: %v", s.ID, s.Operations, s.Requires, s.Required)
}

type Steps []*Step

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
	StateStore store.Store
	// DataStore is used for intermediate data passed between flow nodes
	DataStore store.Store
	// FlowStore is used for storing flows
	FlowStore store.Store
	// EventHandler is used to notification about flow progress
	EventHandler EventHandler
	// PanicHandler is used for recovery panics
	PanicHandler func(interface{})
	// Logger is used internally to provide messages
	Logger logger.Logger
	// Context is used for storing non default options
	Context context.Context
}

type flowKey struct{}

func FlowToContext(ctx context.Context, fl Flow) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, flowKey{}, fl)
}

func FlowFromContext(ctx context.Context) (Flow, error) {
	if ctx == nil {
		return nil, fmt.Errorf("invalid context")
	}
	flow, ok := ctx.Value(flowKey{}).(Flow)
	if !ok {
		return nil, fmt.Errorf("invalid context")
	}
	return flow, nil
}

type ExecuteOptions struct {
	// Passed request id
	ID string
	// Passed step to start swafrom
	Step string
	// Output which step output returns
	Output string
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
	// Client for communication
	Client client.Client
	// Broker for communication
	Broker broker.Broker
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
func WithStateStore(s store.Store) Option {
	return func(o *Options) {
		o.StateStore = s
	}
}

// Data store implementation
func WithDataStore(s store.Store) Option {
	return func(o *Options) {
		o.DataStore = s
	}
}

// Flow store implementation
func WithFlowStore(s store.Store) Option {
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
func WithLogger(l logger.Logger) Option {
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

// Which step output return from flow
func ExecuteOutput(output string) ExecuteOption {
	return func(o *ExecuteOptions) {
		o.Output = output
	}
}

// Step of execution
func ExecuteStep(step string) ExecuteOption {
	return func(o *ExecuteOptions) {
		o.Step = step
	}
}

// ID of execution
func ExecuteID(id string) ExecuteOption {
	return func(o *ExecuteOptions) {
		o.ID = id
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

// Client Call options
func ExecuteClientCallOption(opts ...client.CallOption) ExecuteOption {
	return func(o *ExecuteOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, clientCallOperation{}, opts)
	}
}

// Context for non default options
func ExecuteContext(ctx context.Context) ExecuteOption {
	return func(o *ExecuteOptions) {
		o.Context = ctx
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
