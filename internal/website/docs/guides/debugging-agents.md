---
layout: default
title: Debugging your agent
---

# Debugging your agent

Use this guide when an agent surprises you: it answered without using a service,
called the wrong endpoint, looped, lost memory, refused a tool, or behaved
differently when a flow handed work to it. The local inner loop is:

```sh
micro run          # start services, agents, gateway, dashboard
micro chat         # reproduce one turn
micro inspect ...  # read the recorded run or workflow history
```

Debug the lifecycle in the same order Go Micro runs it: first prove the service is
registered and callable, then inspect the agent run that chose tools, then inspect
any workflow that handed off to the agent. If the first local run fails before a
chat turn, run `micro agent preflight`; failed checks include `Fix:` and `Next:`
lines for Go, CLI installation, provider-key setup, and the local gateway port.

## 1. Reproduce one small turn

Start from the application directory and keep the prompt narrow enough that you
can tell which tool should have run:

```sh
micro run
micro chat --prompt "Create a ticket for Pat, then list open tickets."
```

For a live provider, make the provider choice explicit so a later retry uses the
same model boundary:

```sh
MICRO_AI_PROVIDER=anthropic \
ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" \
micro chat --prompt "Create a ticket for Pat, then list open tickets."
```

If the provider supports streaming, turn it on while you reproduce the issue:

```sh
micro chat --provider anthropic --stream
```

Streaming shows the final answer as it arrives. Tool execution still goes through
the same agent run and is visible through inspection after the turn completes.

## 2. Prove the service side before blaming the model

Agents only call tools that the runtime can discover and describe. Check the
service boundary first:

```sh
micro services
micro call ticket TicketService.List '{}'
```

If the service is missing, restart the service under `micro run` and verify it is
using the same registry as the agent. If the direct `micro call` fails, fix the
handler, request shape, or auth error there before debugging prompts.

When the agent calls the wrong tool or sends the wrong fields, improve the tool
description at the service source:

```go
// Create opens a customer support ticket and returns its stable ticket ID.
// @example {"customer":"Pat","subject":"Cannot log in"}
func (s *TicketService) Create(ctx context.Context, req *CreateRequest, rsp *CreateResponse) error {
```

Endpoint comments, request field names, `description` tags, and `@example` blocks
are the model's map of your service. A vague handler comment often looks like a
reasoning failure from the outside.

## 3. Inspect agent run history

After a chat turn, list recent runs for that agent:

```sh
micro inspect agent support
```

The output shows the run id, status, number of recorded events, the last event,
errors, and a short trace id when tracing is configured. Narrow the list while you
iterate:

```sh
micro inspect agent support --limit 5
micro inspect agent support --status timeout
micro inspect agent support --trace abc123
micro inspect agent support --json
```

Useful statuses include `done`, `refused`, `timeout`, `rate_limited`, `canceled`,
and `error`. Use `--json` when you want exact timestamps, trace/span ids, and error
kinds for a bug report.

Run timelines are stored in the agent's state store under that agent's scoped
state (`agent/<name>/runs/...`). The persisted timeline is recorded even without
an OpenTelemetry exporter, so `micro inspect agent` remains useful in local
no-secret development.

## 4. See tool calls as they happen

When you are embedding an agent in Go and need live tool visibility, use the
streaming API instead of waiting for the final answer:

```go
stream, err := agent.StreamAsk(ctx, ag, "Create a ticket for Pat")
if err != nil {
    return err
}
for {
    ev, err := stream.Recv()
    if err != nil {
        break
    }
    switch ev.Type {
    case agent.StreamEventToolStart:
        log.Printf("tool start: %s %#v", ev.ToolCall.Name, ev.ToolCall.Input)
    case agent.StreamEventToolEnd:
        log.Printf("tool end: %s %#v", ev.ToolCall.Name, ev.Result)
    case agent.StreamEventToken:
        fmt.Print(ev.Token)
    }
}
```

For custom audit logging, wrap the tool execution boundary. Wrappers observe every
call and result, including guardrail refusals:

```go
wrapped := micro.AgentWrapTool(func(next ai.ToolHandler) ai.ToolHandler {
    return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
        if run, ok := ai.RunInfoFrom(ctx); ok {
            log.Printf("run=%s agent=%s tool=%s", run.RunID, run.Agent, call.Name)
        }
        res := next(ctx, call)
        if res.Refused != "" {
            log.Printf("tool refused: %s reason=%s", call.Name, res.Refused)
        }
        return res
    }
})

ag := micro.NewAgent("support", wrapped)
```

Use this when you need request/response payloads in your own logs. By default,
Go Micro records safe run metadata; raw prompt input is not persisted unless the
agent is configured with `agent.TraceInputs(true)`.

## 5. Inspect memory and plans

Default agent memory is store-backed and scoped to the agent name. A restarted
agent with the same `micro.WithStore(...)` and name reloads conversation history
from the `history` key in `agent/<name>` state. If you pass `micro.WithMemory(...)`,
you own that backend; if you pass `agent.NewInMemory(...)`, memory disappears on
restart.

The built-in `plan` tool also saves the current plan to the same scoped agent
state, so a later turn can pick up the saved plan. When memory does not persist,
check that all of these are stable across restarts:

- the agent name (`micro.NewAgent("support", ...)`),
- the configured store backend (`micro.WithStore(...)` or the process default),
- whether a custom in-memory `Memory` implementation replaced the default,
- whether compaction/retrieval limits are intentionally hiding older turns from
  the active model context.

## 6. Inspect workflow handoffs

If a flow triggered the agent, inspect the flow too. The flow history tells you
which durable stage dispatched to the agent and whether a run is still pending:

```sh
micro inspect flow intake
micro inspect flow intake --pending
micro inspect flow intake --stage notify
micro inspect flow intake --json
```

The older flow-specific command remains available for listing runs:

```sh
micro flow runs intake
```

Use the flow run id and the agent run id together when debugging handoffs: the
flow explains why work started and where it checkpointed; the agent run explains
which model/tool steps happened after the handoff.

## 7. Add traces when metadata is not enough

For local CLI debugging, `micro inspect` is the fastest path. For production or
multi-service debugging, configure an OpenTelemetry tracer provider on the agent:

```go
ag := micro.NewAgent("support",
    micro.AgentTraceProvider(tp),
)
```

Trace ids flow into the recorded run summaries, so you can pivot between
`micro inspect agent support --trace <prefix>` and your trace backend. Keep
`agent.TraceInputs(true)` off unless your observability backend is approved to
store prompt content.

## Troubleshooting table

| Symptom | What to inspect | Common fix |
| --- | --- | --- |
| Agent answers without calling a service | `micro services`, direct `micro call`, then `micro inspect agent <name>` | Register the service, include it in `micro.AgentServices(...)`, or improve endpoint comments and examples. |
| Agent loops or burns steps | `micro inspect agent <name> --status error` and wrapper logs | Add or lower `micro.AgentMaxSteps(...)` / `micro.AgentLoopLimit(...)`; move predictable work into a flow. |
| Tool is refused before it runs | Wrapper logs, `ToolResult.Refused`, `micro inspect agent <name> --status refused` | Update `micro.AgentApproveTool(...)` policy or prompt the user for explicit approval before retrying. |
| Memory is missing after restart | Agent name, store backend, `WithMemory`, compaction/retrieval settings | Use the default store-backed memory with a persistent store, or persist your custom memory backend. |
| Flow handoff appears stuck | `micro inspect flow <flow> --pending`, then `micro inspect agent <agent>` | Resume or fail the pending flow run; confirm the dispatched agent completed or timed out. |
| Provider failed or timed out | `micro inspect agent <name> --status timeout` / `--status rate_limited` | Retry with the same provider/model, raise deadlines where appropriate, or enable provider retries for transient errors. |
| Tool call appears as assistant text | Agent run history and provider conformance checks | Keep provider packages current; Go Micro normalizes provider-emitted text tool calls, and conformance tests guard this behavior. |

## What to include in a bug report

When you cannot explain the run locally, include:

```sh
micro inspect agent <agent> --limit 5 --json
micro inspect flow <flow> --limit 5 --json
micro services
micro call <service> <Handler.Method> '{}'
```

Redact secrets and user data. If you enabled `agent.TraceInputs(true)`, inspect the
JSON before sharing it because prompts may be present.
