package agent

import (
	"context"
	"time"

	log "go-micro.dev/v5/logger"
	"go-micro.dev/v5/model"
	"go-micro.dev/v5/registry"

	"go-micro.dev/v5/client"
)

// Options configures an Agent.
type Options struct {
	// Name of the agent.
	Name string

	// Directive is the agent's system prompt describing its purpose
	// and how it should manage the services it is responsible for.
	Directive string

	// Services is the list of service names this agent manages.
	// An empty list means the agent watches all registered services.
	Services []string

	// Model is the AI model the agent uses to reason about service state.
	// When nil the agent still runs and executes its built-in tools but
	// does not perform AI-driven evaluation.
	Model model.Model

	// Registry used to discover and watch services.
	Registry registry.Registry

	// Client used to make RPC calls to services.
	Client client.Client

	// Logger for agent output.
	Logger log.Logger

	// Context for cancellation and deadline propagation.
	Context context.Context

	// Interval between evaluation cycles. Defaults to 30 seconds.
	Interval time.Duration

	// ToolHandler is an optional callback for custom tool execution.
	// It is called when a tool call does not match a built-in tool.
	ToolHandler model.ToolHandler
}

// Option is a function that modifies Options.
type Option func(*Options)

func newOptions(opts ...Option) Options {
	o := Options{
		Name:      "agent",
		Directive: "You are an agent that manages the lifecycle of microservices. Monitor their health and take corrective action when needed.",
		Context:   context.Background(),
		Interval:  30 * time.Second,
		Registry:  registry.DefaultRegistry,
		Client:    client.DefaultClient,
		Logger:    log.DefaultLogger,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// WithName sets the agent name.
func WithName(name string) Option {
	return func(o *Options) {
		o.Name = name
	}
}

// WithDirective sets the agent directive (system prompt).
func WithDirective(directive string) Option {
	return func(o *Options) {
		o.Directive = directive
	}
}

// WithServices sets the list of service names the agent manages.
func WithServices(services ...string) Option {
	return func(o *Options) {
		o.Services = services
	}
}

// WithModel sets the AI model used for evaluation.
func WithModel(m model.Model) Option {
	return func(o *Options) {
		o.Model = m
	}
}

// WithRegistry sets the registry for service discovery.
func WithRegistry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// WithClient sets the RPC client used for service calls.
func WithClient(c client.Client) Option {
	return func(o *Options) {
		o.Client = c
	}
}

// WithLogger sets the logger.
func WithLogger(l log.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}

// WithContext sets the context.
func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

// WithInterval sets the evaluation interval.
func WithInterval(d time.Duration) Option {
	return func(o *Options) {
		o.Interval = d
	}
}

// WithToolHandler sets a custom tool handler for unrecognized tool calls.
func WithToolHandler(h model.ToolHandler) Option {
	return func(o *Options) {
		o.ToolHandler = h
	}
}
