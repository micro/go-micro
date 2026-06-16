# Agent Tool Wrappers

Middleware around an agent's tool execution, the same way
`client.CallWrapper` and `server.HandlerWrapper` wrap RPCs.

Every tool call an agent makes runs through `ai.ToolHandler`:

```go
type ToolHandler func(ctx context.Context, call ai.ToolCall) ai.ToolResult
type ToolWrapper func(ai.ToolHandler) ai.ToolHandler
```

`WrapTool` (exposed as `micro.AgentWrapTool`) registers a wrapper: it
takes the next handler and returns a new one. Code before `next(...)`
runs before the tool, code after runs after. That single seam covers
the whole lifecycle — before/after hooks, timing, metrics, retries,
inspecting results.

## What this example does

One flaky `weather` service and one agent with two wrappers:

- **observe** — times every call and records a per-tool count, logging
  the correlation ID (`call.ID`) carried through from the provider. It
  observes; it changes nothing.
- **retry** — re-runs a call whose result is an error, up to three
  attempts. The weather service fails the first time it's hit and
  succeeds after, so retry turns a transient failure into a success the
  model never sees.

Wrappers compose **outermost-first**: `observe` is registered first, so
it wraps `retry` and sees one logical call even when retry runs the tool
twice.

```go
micro.NewAgent("forecaster",
    micro.AgentServices("weather"),
    micro.AgentProvider(provider),
    micro.AgentAPIKey(apiKey),
    micro.AgentWrapTool(m.observe, retry(3)),
)
```

## Wrappers vs. guardrails

Developer wrappers run **outside** the built-in guardrails (`MaxSteps`,
`LoopLimit`, `ApproveTool`), so they see every call and its result —
including a guardrail's refusal. The flip side: a retry wrapper's
`next` is the full guardrail stack, so each retry is also counted by
loop detection. Keep `LoopLimit` at or above your retry count, or set
`AgentLoopLimit(0)` when a wrapper owns the repetition.

See the [Agent Guardrails guide](../../internal/website/docs/guides/agent-guardrails.md)
for the full picture.

## Run

Needs an LLM provider key:

```bash
export ANTHROPIC_API_KEY=sk-ant-...   # or OPENAI_API_KEY, GEMINI_API_KEY, ...
go run main.go
```
