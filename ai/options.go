package ai

import (
	"context"
	"net/http"
)

// Options for model configuration
type Options struct {
	// Context for the model
	Context context.Context
	// Model name (e.g., "gpt-4o", "claude-sonnet-4-20250514")
	Model string
	// APIKey for authentication
	APIKey string
	// BaseURL for the API endpoint
	BaseURL string
	// ToolHandler handles tool calls (optional, for automatic tool execution)
	ToolHandler ToolHandler
	// MaxTokens caps the length of the response (0 = provider default)
	MaxTokens int
	// Thinking controls Anthropic's extended-thinking mode. Empty leaves the
	// provider default in place.
	Thinking ThinkingMode
	// Effort controls reasoning depth for providers that support it.
	Effort string
	// Transport is the HTTP round tripper used by providers that make live
	// API calls. Nil uses the standard library default transport. Inject a
	// fake for tests.
	Transport http.RoundTripper

	// NoCache disables prompt-prefix caching for providers that support it
	// (e.g. Anthropic cache_control). Caching is on by default because the
	// dominant caller — the agent tool loop — re-sends an identical prefix on
	// every round, repaying the cache write within a single Generate. A
	// one-off caller whose prompt or tools change every request can opt out,
	// since cache writes bill at a premium over plain input.
	NoCache bool
}

// ThinkingMode controls whether a reasoning-capable model uses extended thinking.
type ThinkingMode string

const (
	ThinkingAdaptive ThinkingMode = "adaptive"
	ThinkingDisabled ThinkingMode = "disabled"
)

// GenerateOptions for generate call
type GenerateOptions struct {
	// Context for this specific generate call
	Context context.Context
}

// Option is a function that modifies Options
type Option func(*Options)

// GenerateOption is a function that modifies GenerateOptions
type GenerateOption func(*GenerateOptions)

// NewOptions creates new Options with defaults
func NewOptions(opts ...Option) Options {
	options := Options{
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}
	return options
}

// WithModel sets the model name
func WithModel(m string) Option {
	return func(o *Options) {
		o.Model = m
	}
}

// WithAPIKey sets the API key
func WithAPIKey(key string) Option {
	return func(o *Options) {
		o.APIKey = key
	}
}

// WithBaseURL sets the base URL
func WithBaseURL(url string) Option {
	return func(o *Options) {
		o.BaseURL = url
	}
}

// WithContext sets the context
func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

// WithToolHandler sets the tool handler
func WithToolHandler(handler ToolHandler) Option {
	return func(o *Options) {
		o.ToolHandler = handler
	}
}

// WithTools wires a Tools instance into the model, setting the tool
// handler so the model can execute discovered service endpoints. The
// tool list itself is passed per-request via Request.Tools.
//
//	tools := ai.NewTools(service.Registry())
//	list, _ := tools.Discover()
//	m := ai.New("anthropic", ai.WithAPIKey(key), ai.WithTools(tools))
//	resp, _ := m.Generate(ctx, &ai.Request{Prompt: input, Tools: list})
func WithTools(t *Tools) Option {
	return func(o *Options) {
		if t != nil {
			o.ToolHandler = t.Handler()
		}
	}
}

// WithMaxTokens caps the number of tokens in the response. 0 leaves the
// provider default in place.
func WithMaxTokens(n int) Option {
	return func(o *Options) {
		o.MaxTokens = n
	}
}

// WithThinking sets the extended-thinking mode for providers that support it.
func WithThinking(mode ThinkingMode) Option {
	return func(o *Options) {
		o.Thinking = mode
	}
}

// WithEffort sets the reasoning effort for providers that support it. An empty
// value leaves the provider default in place.
// WithoutCache disables prompt-prefix caching for providers that support it.
func WithoutCache() Option {
	return func(o *Options) {
		o.NoCache = true
	}
}

func WithEffort(effort string) Option {
	return func(o *Options) {
		o.Effort = effort
	}
}

// WithTransport sets the HTTP round tripper used for provider API calls.
// Nil uses the standard library default transport. Inject a fake
// RoundTripper in tests to stub provider responses without a live server.
func WithTransport(rt http.RoundTripper) Option {
	return func(o *Options) {
		o.Transport = rt
	}
}
