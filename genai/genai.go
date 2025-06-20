// Package genai provides a generic interface for generative AI providers.
package genai

// Result is the unified response from GenAI providers.
type Result struct {
	Prompt string
	Type   string
	Data   []byte // for audio/image binary data
	Text   string // for text or image URL
}

// Stream represents a streaming response from a GenAI provider.
type Stream struct {
	Results <-chan *Result
	Err     error
	// You can add fields for cancellation, errors, etc. if needed
}

// GenAI is the generic interface for generative AI providers.
type GenAI interface {
	Generate(prompt string, opts ...Option) (*Result, error)
	Stream(prompt string, opts ...Option) (*Stream, error)
}

// Option is a functional option for configuring providers.
type Option func(*Options)

// Options holds configuration for providers.
type Options struct {
	APIKey   string
	Endpoint string
	Type     string // "text", "image", "audio", etc.
	Model    string // model name, e.g. "gemini-2.5-pro"
	// Add more fields as needed
}

// Option functions for generation type
func Text(o *Options)  { o.Type = "text" }
func Image(o *Options) { o.Type = "image" }
func Audio(o *Options) { o.Type = "audio" }

// Provider registry
var providers = make(map[string]GenAI)

// Register a GenAI provider by name.
func Register(name string, provider GenAI) {
	providers[name] = provider
}

// Get a GenAI provider by name.
func Get(name string) GenAI {
	return providers[name]
}
