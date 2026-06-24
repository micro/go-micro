package agent

import (
	"context"
	"time"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
	"go.opentelemetry.io/otel/trace"
)

// Option configures an Agent.
type Option func(*Options)

// ApproveFunc decides whether an agent may execute a tool call before it
// runs. Returning false blocks the call; the reason is shown to the
// model so it can adapt. Use it for human-in-the-loop approval or policy
// checks. It is called for actions (service tools and delegate), not for
// the internal plan tool.
type ApproveFunc func(tool string, input map[string]any) (approved bool, reason string)

// ToolFunc handles a custom tool call. Return the result as a string
// (often JSON); return an error to report failure back to the model.
type ToolFunc func(ctx context.Context, input map[string]any) (string, error)

// customTool is a developer-registered tool beyond the agent's services.
type customTool struct {
	def     ai.Tool
	handler ToolFunc
}

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

	// ModelTimeout bounds each provider Generate call (0 disables).
	ModelTimeout time.Duration
	// ModelMaxAttempts bounds provider Generate attempts including the first
	// call. Default 1 — retries are opt-in (enable with ModelRetry). A Generate
	// runs the whole tool-execution turn, so auto-retrying it would re-run
	// already-executed, possibly side-effecting tool calls; keep it explicit.
	ModelMaxAttempts int
	// ModelRetryBackoff is the base delay between transient provider failures
	// (grows exponentially per attempt when retries are enabled).
	ModelRetryBackoff time.Duration

	// Memory is the agent's conversation memory. Nil = the default
	// store-backed memory (durable across restarts).
	Memory Memory

	// MaxSteps bounds the number of tool executions per Ask (0 =
	// unbounded). Once exceeded, further tool calls are refused and the
	// model is told to stop and summarize. A stopping condition.
	MaxSteps int
	// LoopLimit bounds how many times the agent may call the same tool
	// with the same arguments in one Ask before the call is refused as a
	// no-progress loop (0 = disabled). Catches the agent repeating an
	// identical action — which MaxSteps only bounds by total count.
	LoopLimit int
	// Approve gates each action before it runs. Nil = allow all.
	Approve ApproveFunc

	// A2AAddress, if set, makes Run serve this agent over the A2A protocol
	// on that address directly (no separate gateway), e.g. ":4000".
	A2AAddress string

	// TraceProvider enables OpenTelemetry spans for agent runs, model calls,
	// and tool calls. Nil disables instrumentation.
	TraceProvider trace.TracerProvider

	// tools are developer-registered custom tools (see WithTool).
	tools []customTool
	// wrappers are developer-registered tool-execution wrappers
	// (see WrapTool), applied outside the built-in guardrails.
	wrappers []ai.ToolWrapper
}

func newOptions(opts ...Option) Options {
	o := Options{
		Registry:          registry.DefaultRegistry,
		Client:            client.DefaultClient,
		Store:             store.DefaultStore,
		HistoryLimit:      50,
		ModelTimeout:      30 * time.Second,
		ModelMaxAttempts:  1, // retries opt-in via ModelRetry (see field doc)
		ModelRetryBackoff: 100 * time.Millisecond,
		// On by default and lenient: identical repeated calls are a
		// no-progress loop, never useful. Set LoopLimit(0) to disable.
		LoopLimit: 3,
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

// LoopLimit sets how many times the agent may repeat the same tool call
// (same name and arguments) in one Ask before it is refused as a
// no-progress loop. 0 disables loop detection.
func LoopLimit(n int) Option {
	return func(o *Options) { o.LoopLimit = n }
}

// ModelCallTimeout sets the timeout for each provider Generate call.
func ModelCallTimeout(d time.Duration) Option {
	return func(o *Options) { o.ModelTimeout = d }
}

// ModelRetry sets the provider retry budget and backoff for transient failures.
func ModelRetry(maxAttempts int, backoff time.Duration) Option {
	return func(o *Options) {
		o.ModelMaxAttempts = maxAttempts
		o.ModelRetryBackoff = backoff
	}
}

// WithA2A makes Run serve the agent over the A2A protocol on addr (e.g.
// ":4000"), so other agents can reach it directly by URL without a
// separate gateway. The agent stays a normal go-micro service as well;
// this adds a second, A2A-native HTTP endpoint that calls it in-process.
func WithA2A(addr string) Option {
	return func(o *Options) { o.A2AAddress = addr }
}

// WithMemory sets the agent's conversation memory. The default is
// store-backed memory keyed by agent name; supply your own to use an
// in-process, database, or semantic store.
func WithMemory(m Memory) Option {
	return func(o *Options) { o.Memory = m }
}

// WrapTool registers a tool-execution wrapper, the tool-side analog of
// a client/server middleware wrapper. Each wrapper takes the next handler
// and returns a new one; code before the next(...) call runs before the
// tool executes, code after runs after. Use it for logging, metrics,
// retries, or custom policy. Wrappers run outside the built-in guardrails
// (MaxSteps, LoopLimit, ApproveTool), so they observe every call and its
// result, including refusals. Multiple wrappers compose outermost-first.
//
//	micro.NewAgent("worker", micro.AgentWrapTool(
//	    func(next ai.ToolHandler) ai.ToolHandler {
//	        return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
//	            res := next(ctx, call)
//	            log.Printf("id=%s tool=%s", call.ID, call.Name)
//	            return res
//	        }
//	    }))
func WrapTool(w ...ai.ToolWrapper) Option {
	return func(o *Options) {
		o.wrappers = append(o.wrappers, w...)
	}
}

// WithTool registers a custom tool the agent can call, beyond the
// services it discovers — a local function, an external API, anything.
// properties is the JSON-schema map for the tool's parameters.
func WithTool(name, description string, properties map[string]any, handler ToolFunc) Option {
	return func(o *Options) {
		o.tools = append(o.tools, customTool{
			def: ai.Tool{
				Name:         name,
				OriginalName: name,
				Description:  description,
				Properties:   properties,
			},
			handler: handler,
		})
	}
}

// TraceProvider enables OpenTelemetry tracing for agent runs. When nil,
// agent tracing and run timeline recording are disabled.
func TraceProvider(tp trace.TracerProvider) Option {
	return func(o *Options) { o.TraceProvider = tp }
}
