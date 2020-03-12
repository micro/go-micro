package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/store"
)

type Option func(*Options)

type Options struct {
	// Number of worker goroutines
	//Concurrency int
	// Preallocate worker goroutines
	//Prealloc bool
	// If no workers available no blocking
	//Nonblock bool
	// Wait completiong before stop
	//Wait bool
	// Executor to run flow
	Executor Executor
	// Store generic store to use for all things
	//Store store.Store
	// StateStore is used for flow state marking
	//StateStore store.Store
	// DataStore is used for intermediate data passed between flow nodes
	//DataStore store.Store
	// FlowStore is used for storing flows
	//FlowStore store.Store
	// ErrorHandler is used for recovery panics
	ErrorHandler func(interface{})
	// Logger is used internally to provide messages
	//Logger logger.Logger
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
	// Passed flow name
	Flow string
	// Passed request id
	ID string
	// Passed step to start from
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
	// Client for communication
	Client client.Client
	// Context is used for storing non default options
	Context context.Context
}

type ExecuteOption func(*ExecuteOptions)

// Pass executor
func WithExecutor(exe Executor) Option {
	return func(o *Options) {
		o.Executor = exe
	}
}

type waitOptionKey struct{}

// Wait for flow completion before stop
func WithWait(b bool) Option {
	return setOption(waitOptionKey{}, b)
}

type nonblockOptionKey struct{}

// Nonblocking submission
func WithNonblock(b bool) Option {
	return setOption(nonblockOptionKey{}, b)
}

// Panic handler
func WithErrorHandler(h func(interface{})) Option {
	return func(o *Options) {
		o.ErrorHandler = h
	}
}

type preallocOptionKey struct{}

// WithPrealloc preallocates goroutine pool
func WithPrealloc(b bool) Option {
	return setOption(preallocOptionKey{}, b)
}

type concurrencyOptionKey struct{}

// Size of goroutine pool
func WithConcurrency(c int) Option {
	return setOption(concurrencyOptionKey{}, c)
}

type storeOptionKey struct{}

// Store to be used for all flow operations
func WithStore(s store.Store) Option {
	return setOption(storeOptionKey{}, s)
}

type stateStoreOptionKey struct{}

// State store implementation
func WithStateStore(s store.Store) Option {
	return setOption(stateStoreOptionKey{}, s)
}

type dataStoreOptionKey struct{}

// Data store implementation
func WithDataStore(s store.Store) Option {
	return setOption(dataStoreOptionKey{}, s)
}

type flowStoreOptionKey struct{}

// Flow store implementation
func WithFlowStore(s store.Store) Option {
	return setOption(flowStoreOptionKey{}, s)
}

type loggerOptionKey struct{}

// Logger
func WithLogger(l logger.Logger) Option {
	return setOption(loggerOptionKey{}, l)
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
