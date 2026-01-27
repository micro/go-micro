// Package genai provides a generic interface for generative AI providers.
package genai

import (
	"context"
	"errors"
	"sync"
)

var (
	// ErrNoProvider is returned when no GenAI provider is configured.
	ErrNoProvider = errors.New("no genai provider configured")
)

// Result is the unified response from GenAI providers.
type Result struct {
	Prompt string
	Type   string
	Data   []byte // for audio/image binary data
	Text   string // for text or image URL
	Error  error  // error if this chunk failed
}

// Stream represents a streaming response from a GenAI provider.
type Stream struct {
	Results <-chan *Result
	cancel  context.CancelFunc
}

// Close cancels the stream and releases resources.
func (s *Stream) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}

// NewStream creates a new stream with the given channel and cancel function.
func NewStream(results <-chan *Result, cancel context.CancelFunc) *Stream {
	return &Stream{
		Results: results,
		cancel:  cancel,
	}
}

// GenAI is the generic interface for generative AI providers.
type GenAI interface {
	// Generate performs a single request and returns the result.
	Generate(ctx context.Context, prompt string, opts ...Option) (*Result, error)
	// Stream performs a streaming request and returns results as they arrive.
	Stream(ctx context.Context, prompt string, opts ...Option) (*Stream, error)
	// String returns the provider name.
	String() string
}

// Option is a functional option for configuring providers.
type Option func(*Options)

// Options holds configuration for providers.
type Options struct {
	APIKey      string
	Endpoint    string
	Type        string        // "text", "image", "audio", etc.
	Model       string        // model name, e.g. "gemini-2.5-pro"
	MaxTokens   int           // maximum tokens to generate
	Temperature float64       // sampling temperature (0.0-2.0)
	Timeout     int           // request timeout in seconds
}

// WithAPIKey sets the API key.
func WithAPIKey(key string) Option {
	return func(o *Options) { o.APIKey = key }
}

// WithEndpoint sets a custom endpoint URL.
func WithEndpoint(endpoint string) Option {
	return func(o *Options) { o.Endpoint = endpoint }
}

// WithModel sets the model name.
func WithModel(model string) Option {
	return func(o *Options) { o.Model = model }
}

// WithMaxTokens sets the maximum tokens to generate.
func WithMaxTokens(tokens int) Option {
	return func(o *Options) { o.MaxTokens = tokens }
}

// WithTemperature sets the sampling temperature.
func WithTemperature(temp float64) Option {
	return func(o *Options) { o.Temperature = temp }
}

// WithTimeout sets the request timeout in seconds.
func WithTimeout(seconds int) Option {
	return func(o *Options) { o.Timeout = seconds }
}

// Type option functions
func Text(o *Options)  { o.Type = "text" }
func Image(o *Options) { o.Type = "image" }
func Audio(o *Options) { o.Type = "audio" }

// Provider registry with thread-safe access
var (
	providers   = make(map[string]GenAI)
	providersMu sync.RWMutex
)

// Register a GenAI provider by name.
func Register(name string, provider GenAI) {
	providersMu.Lock()
	defer providersMu.Unlock()
	providers[name] = provider
}

// Get a GenAI provider by name.
func Get(name string) GenAI {
	providersMu.RLock()
	defer providersMu.RUnlock()
	return providers[name]
}

// List returns all registered provider names.
func List() []string {
	providersMu.RLock()
	defer providersMu.RUnlock()
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}
