// Package ai provides abstraction for AI model providers
package ai

import (
	"context"
	"encoding/json"
	"errors"
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
	Name         string // LLM-safe name (e.g., "greeter_Greeter_Hello")
	OriginalName string // Original name (e.g., "greeter.Greeter.Hello")
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
	// Messages for continuing a conversation (optional).
	// Use ai.History to accumulate these across turns.
	Messages []Message
}

// Message represents a conversation message
type Message struct {
	Role    string // "user", "assistant", "system", "tool"
	Content any    // Can be string or structured content
}

// Usage describes token counts returned by model providers.
type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
}

// Response represents the response from a model
type Response struct {
	// Reply is the text response from the model
	Reply string
	// ToolCalls are tool calls requested by the model
	ToolCalls []ToolCall
	// Answer is the final answer after tool execution (if tools were used)
	Answer string
	// Usage contains provider token usage when available.
	Usage Usage
}

// ToolCall represents a request to call a tool and its result
type ToolCall struct {
	ID     string         // Tool call ID (for correlation)
	Name   string         // Tool name
	Input  map[string]any // Tool input arguments
	Result string         // Tool execution result (populated after execution)
	Error  string         // Tool execution error (populated after execution)
}

// Scan decodes the call's Input into v (a pointer to a struct or map),
// the same way a codec decodes an RPC request body. Use it when a tool
// wants typed arguments instead of the raw map:
//
//	var args struct{ Query string `json:"query"` }
//	if err := call.Scan(&args); err != nil { ... }
func (c ToolCall) Scan(v any) error {
	b, err := json.Marshal(c.Input)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ID      string // Tool call ID (for correlation)
	Value   any    // Structured result (optional)
	Content string // Tool execution result (JSON string), shown to the model
	// Refused names the reason a guardrail blocked the call before it ran
	// ("max_steps", "loop", "approval"); empty when the call executed. A
	// tool wrapper can switch on it to build reliability tooling — react to
	// a detected loop, audit refusals — without parsing the message.
	Refused string `json:"refused,omitempty"`
}

// Refusal reason codes set on ToolResult.Refused by the agent's guardrails.
const (
	RefusedMaxSteps = "max_steps"
	RefusedLoop     = "loop"
	RefusedApproval = "approval"
)

// RunInfo describes the agent run a tool call belongs to. The agent
// attaches it to the context passed to a ToolHandler, so a wrapper can
// correlate calls within a run and across delegation without coupling to
// the agent package. Per-call detail (tool name, id) is on the ToolCall;
// step and attempt counts are naturally counted by the wrapper itself.
type RunInfo struct {
	RunID    string // correlation id for this agent run (one per Ask)
	ParentID string // the run that delegated to this one, if any
	Agent    string // the agent's name
}

type runInfoKey struct{}

// WithRunInfo attaches run info to ctx.
func WithRunInfo(ctx context.Context, r RunInfo) context.Context {
	return context.WithValue(ctx, runInfoKey{}, r)
}

// RunInfoFrom returns the run info attached to ctx, and whether it was set.
func RunInfoFrom(ctx context.Context) (RunInfo, bool) {
	r, ok := ctx.Value(runInfoKey{}).(RunInfo)
	return r, ok
}

// ErrStreamingUnsupported is returned by providers that implement the Model
// interface but do not yet support token streaming. Use errors.Is so callers
// can distinguish an unsupported capability from transient provider failures.
var ErrStreamingUnsupported = errors.New("ai: streaming unsupported")

// Stream is the interface for streaming responses.
type Stream interface {
	// Recv receives the next chunk of the response
	Recv() (*Response, error)
	// Close closes the stream
	Close() error
}

// ToolHandler executes a tool call and returns its result. It mirrors a
// go-micro RPC handler — context first, a request in, a result out — so
// the same mental model carries over from services to tools.
type ToolHandler func(ctx context.Context, call ToolCall) ToolResult

// ToolWrapper wraps a ToolHandler to add behavior around execution —
// logging, metrics, retries, guardrails. It is the tool-side analog of
// client.CallWrapper and server.HandlerWrapper: a wrapper takes the next
// handler and returns a new one, and code before the next(...) call runs
// before the tool, code after runs after.
type ToolWrapper func(ToolHandler) ToolHandler

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
	switch {
	case strings.Contains(baseURL, "anthropic"):
		return "anthropic"
	case strings.Contains(baseURL, "atlascloud"):
		return "atlascloud"
	case strings.Contains(baseURL, "googleapis.com"), strings.Contains(baseURL, "google"):
		return "gemini"
	case strings.Contains(baseURL, "groq"):
		return "groq"
	case strings.Contains(baseURL, "mistral"):
		return "mistral"
	case strings.Contains(baseURL, "together"):
		return "together"
	default:
		return "openai"
	}
}

// DefaultModel is a default model instance
var DefaultModel Model

// Generate generates a response using the default model.
func Generate(ctx context.Context, req *Request, opts ...GenerateOption) (*Response, error) {
	if DefaultModel == nil {
		return nil, nil
	}
	return DefaultModel.Generate(ctx, req, opts...)
}
