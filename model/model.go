// Package model provides abstraction for AI model providers
package model

import (
	"encoding/json"
	"strings"
)

// Model provides an interface for interacting with AI model providers
type Model interface {
	// Name returns the provider name (e.g., "anthropic", "openai")
	Name() string
	// DefaultModel returns the default model name for this provider
	DefaultModel() string
	// DefaultBaseURL returns the default API base URL for this provider
	DefaultBaseURL() string
	// BuildRequest constructs a request payload for the provider's API
	BuildRequest(prompt string, systemPrompt string, tools []Tool, messages []Message) ([]byte, error)
	// ParseResponse parses the provider's API response
	ParseResponse(body []byte) (*Response, error)
	// BuildFollowUpRequest constructs a follow-up request with tool results
	BuildFollowUpRequest(prompt string, systemPrompt string, originalResponse *Response, toolResults []ToolResult) ([]byte, error)
	// ParseFollowUpResponse parses the follow-up response
	ParseFollowUpResponse(body []byte) (string, error)
	// SetAuthHeaders sets the required authentication headers for the provider
	SetAuthHeaders(headers map[string]string, apiKey string)
	// GetAPIEndpoint returns the full API endpoint URL
	GetAPIEndpoint(baseURL string) string
}

// Tool represents a tool/function that can be called by the model
type Tool struct {
	Name        string         // LLM-safe name (e.g., "greeter_Greeter_Hello")
	OriginalName string        // Original name (e.g., "greeter.Greeter.Hello")
	Description string
	Properties  map[string]any // JSON schema for tool parameters
}

// Message represents a conversation message
type Message struct {
	Role    string // "user", "assistant", "system", "tool"
	Content any    // Can be string or structured content
}

// Response represents the parsed response from a model
type Response struct {
	Reply     string      // Text reply from the model
	ToolCalls []ToolCall  // Tool calls requested by the model
	RawContent any        // Provider-specific raw content for follow-up requests
}

// ToolCall represents a request to call a tool
type ToolCall struct {
	ID    string         // Tool call ID (for correlation)
	Name  string         // Tool name
	Input map[string]any // Tool input arguments
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ID      string // Tool call ID (for correlation)
	Content string // Tool execution result (JSON string)
}

// Config holds configuration for a model provider
type Config struct {
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
}

// ProviderFactory is a function that creates a Model instance
type ProviderFactory func(Options) Model

var providers = make(map[string]ProviderFactory)

// Register registers a provider factory
func Register(name string, factory ProviderFactory) {
	providers[name] = factory
}

// New creates a new Model instance based on the provider name
func New(provider string, opts ...Option) (Model, error) {
	options := newOptions(opts...)
	
	if factory, ok := providers[provider]; ok {
		return factory(options), nil
	}
	
	// Default to first registered provider or nil
	if len(providers) > 0 {
		for _, factory := range providers {
			return factory(options), nil
		}
	}
	
	return nil, nil
}

// AutoDetectProvider attempts to detect the provider from the base URL
func AutoDetectProvider(baseURL string) string {
	if baseURL == "" {
		return "openai"
	}
	// Simple detection based on URL
	if strings.Contains(baseURL, "anthropic") {
		return "anthropic"
	}
	return "openai"
}

// mapGoTypeToJSON is a helper to convert Go types to JSON schema types
func mapGoTypeToJSON(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "string"
	}
}

// UnmarshalJSON is a helper for unmarshaling JSON
func UnmarshalJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// MarshalJSON is a helper for marshaling JSON
func MarshalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}
