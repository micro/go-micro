// Package model provides abstraction for AI model providers
package model

import (
	"context"
	"strings"
)

// Model provides an interface for interacting with AI model providers
type Model interface {
	// Init initializes the model with options
	Init(...Option) error
	// Options returns the model options
	Options() Options
	// Generate generates a response from the model
	Generate(ctx context.Context, req *Request, opts ...GenerateOption) (*Response, error)
	// Stream generates a streaming response (for future implementation)
	Stream(ctx context.Context, req *Request, opts ...GenerateOption) (Stream, error)
	// String returns the name of the provider
	String() string
}

// Tool represents a tool/function that can be called by the model
type Tool struct {
	Name         string         // LLM-safe name (e.g., "greeter_Greeter_Hello")
	OriginalName string         // Original name (e.g., "greeter.Greeter.Hello")
	Description  string
	Properties   map[string]any // JSON schema for tool parameters
}

// Request represents a request to generate content from a model
type Request struct {
	// Prompt is the user's message/prompt
	Prompt string
	// SystemPrompt is the system instruction for the model
	SystemPrompt string
	// Tools available for the model to use
	Tools []Tool
	// Messages for continuing a conversation (optional)
	Messages []Message
}

// Message represents a conversation message
type Message struct {
	Role    string // "user", "assistant", "system", "tool"
	Content any    // Can be string or structured content
}

// Response represents the response from a model
type Response struct {
	// Reply is the text response from the model
	Reply string
	// ToolCalls are tool calls requested by the model
	ToolCalls []ToolCall
	// Answer is the final answer after tool execution (if tools were used)
	Answer string
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

// Stream is the interface for streaming responses (future implementation)
type Stream interface {
	// Recv receives the next chunk of the response
	Recv() (*Response, error)
	// Close closes the stream
	Close() error
}

// ToolHandler is a function that handles tool calls
type ToolHandler func(name string, input map[string]any) (result any, content string)

// NewFunc creates a new Model instance
type NewFunc func(...Option) Model

var providers = make(map[string]NewFunc)

// Register registers a model provider
func Register(name string, fn NewFunc) {
	providers[name] = fn
}

// New creates a new Model instance based on the provider name
func New(provider string, opts ...Option) Model {
	if fn, ok := providers[provider]; ok {
		return fn(opts...)
	}
	
	// Default to first registered provider
	if len(providers) > 0 {
		for _, fn := range providers {
			return fn(opts...)
		}
	}
	
	return nil
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

// DefaultModel is a default model instance
var DefaultModel Model

// Generate generates a response using the default model
func Generate(ctx context.Context, req *Request, opts ...GenerateOption) (*Response, error) {
	if DefaultModel == nil {
		return nil, nil
	}
	return DefaultModel.Generate(ctx, req, opts...)
}
