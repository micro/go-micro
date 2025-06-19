// Package genai provides a generic interface for generative AI providers.
package genai

// GenAI is the generic interface for generative AI providers.
type GenAI interface {
	GenerateText(prompt string, opts ...Option) (string, error)
	GenerateImage(prompt string, opts ...Option) (string, error)
	SpeechToText(audioData []byte, opts ...Option) (string, error)
}

// Option is a functional option for configuring providers.
type Option func(*Options)

// Options holds configuration for providers.
type Options struct {
	APIKey   string
	Endpoint string
	// Add more fields as needed
}

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
