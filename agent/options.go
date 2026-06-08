package agent

import (
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/store"
)

// Option configures an Agent.
type Option func(*Options)

// ApproveFunc decides whether an agent may execute a tool call before it
// runs. Returning false blocks the call; the reason is shown to the
// model so it can adapt. Use it for human-in-the-loop approval or policy
// checks. It is called for actions (service tools and delegate), not for
// the internal plan tool.
type ApproveFunc func(tool string, input map[string]any) (approved bool, reason string)

// Options holds agent configuration.
type Options struct {
	Name         string
	Services     []string
	Prompt       string
	Provider     string
	Model        string
	APIKey       string
	Registry     registry.Registry
	Client       client.Client
	Store        store.Store
	HistoryLimit int

	// MaxSteps bounds the number of tool executions per Ask (0 =
	// unbounded). Once exceeded, further tool calls are refused and the
	// model is told to stop and summarize. A stopping condition.
	MaxSteps int
	// Approve gates each action before it runs. Nil = allow all.
	Approve ApproveFunc
}

func newOptions(opts ...Option) Options {
	o := Options{
		Registry:     registry.DefaultRegistry,
		Client:       client.DefaultClient,
		Store:        store.DefaultStore,
		HistoryLimit: 50,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// Name sets the agent name.
func Name(n string) Option {
	return func(o *Options) { o.Name = n }
}

// Services sets which services this agent manages.
func Services(names ...string) Option {
	return func(o *Options) { o.Services = names }
}

// Prompt sets the system prompt.
func Prompt(p string) Option {
	return func(o *Options) { o.Prompt = p }
}

// Provider sets the LLM provider.
func Provider(p string) Option {
	return func(o *Options) { o.Provider = p }
}

// Model sets the LLM model name.
func Model(m string) Option {
	return func(o *Options) { o.Model = m }
}

// APIKey sets the API key for the LLM provider.
func APIKey(k string) Option {
	return func(o *Options) { o.APIKey = k }
}

// WithRegistry sets the service registry.
func WithRegistry(r registry.Registry) Option {
	return func(o *Options) { o.Registry = r }
}

// WithClient sets the RPC client.
func WithClient(c client.Client) Option {
	return func(o *Options) { o.Client = c }
}

// WithStore sets the store for agent memory.
func WithStore(s store.Store) Option {
	return func(o *Options) { o.Store = s }
}

// HistoryLimit sets the max conversation messages to retain.
func HistoryLimit(n int) Option {
	return func(o *Options) { o.HistoryLimit = n }
}

// MaxSteps bounds tool executions per Ask (0 = unbounded). A stopping
// condition: beyond the limit, tool calls are refused and the model is
// told to stop and summarize.
func MaxSteps(n int) Option {
	return func(o *Options) { o.MaxSteps = n }
}

// ApproveTool sets a human-in-the-loop / policy hook called before each
// action (service tools and delegate). Returning false blocks the call.
func ApproveTool(fn ApproveFunc) Option {
	return func(o *Options) { o.Approve = fn }
}
