package agent

import (
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/store"
)

// Option configures an Agent.
type Option func(*Options)

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
	Broker       broker.Broker
	HistoryLimit int
}

func newOptions(opts ...Option) Options {
	o := Options{
		Registry:     registry.DefaultRegistry,
		Client:       client.DefaultClient,
		Store:        store.DefaultStore,
		Broker:       broker.DefaultBroker,
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

// Prompt sets the system prompt — the agent's identity and domain knowledge.
func Prompt(p string) Option {
	return func(o *Options) { o.Prompt = p }
}

// Provider sets the LLM provider (anthropic, openai, etc.).
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

// WithBroker sets the broker for agent-to-agent communication.
func WithBroker(b broker.Broker) Option {
	return func(o *Options) { o.Broker = b }
}

// HistoryLimit sets the max conversation messages to retain.
func HistoryLimit(n int) Option {
	return func(o *Options) { o.HistoryLimit = n }
}
