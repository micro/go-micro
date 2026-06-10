# Agent Interface Design

## Principle

Service = capability. Agent = intelligence. An agent IS a service — it has a real RPC server, a proto-defined `Agent.Chat` endpoint, and registers in the registry like everything else.

```
micro.New("task")           // creates a service
micro.NewAgent("task-mgr")  // creates an agent (which is also a service)
```

Same package. Same level. Same communication (RPC). Different responsibilities.

## Interface

```go
type Agent interface {
    Name() string
    Init(...AgentOption)
    Options() AgentOptions
    Ask(ctx context.Context, message string) (*Response, error)
    Run() error
    Stop() error
    String() string
}
```

**Ask** is the programmatic API. Send a message, get a response.

**Run** starts a real RPC server, registers the `Agent.Chat` endpoint in the registry, and blocks.

## Proto Definition

```protobuf
service Agent {
    rpc Chat(ChatRequest) returns (ChatResponse) {}
}

message ChatRequest {
    string message = 1;
}

message ChatResponse {
    string reply = 1;
    string agent = 2;
    repeated ToolCall tool_calls = 3;
}
```

The agent is callable by any go-micro client:

```bash
micro call task-mgr Agent.Chat '{"message": "What tasks are overdue?"}'
```

## Options

```go
type AgentOptions struct {
    Name         string
    Services     []string          // which services this agent manages
    Prompt       string            // system prompt — identity, domain knowledge, boundaries
    Provider     string            // LLM provider (anthropic, openai, etc.)
    Model        string            // LLM model (optional)
    APIKey       string
    Registry     registry.Registry // discover services and other agents
    Client       client.Client     // call service endpoints and other agents
    Store        store.Store       // backing store for the default memory
    HistoryLimit int               // max conversation turns to retain
    Memory       Memory            // pluggable conversation memory (default: store-backed)
    MaxSteps     int               // stopping condition: max tool calls per Ask
    Approve      ApproveFunc        // human-in-the-loop / policy gate on each action
}
```

## Pluggable composition

An agent composes the same way a service does — a small set of pluggable pieces with working defaults:

| Piece | Default | Swap with |
|-------|---------|-----------|
| **Model** | first registered provider | `AgentProvider` / `AgentModel` |
| **Memory** | store-backed, durable across restarts | `AgentMemory(m Memory)` |
| **Tools** | the agent's services (RPC) + `plan`/`delegate` | `AgentTool(name, desc, schema, fn)` for any function |
| **Guardrails** | none | `AgentMaxSteps`, `AgentApproveTool` |

```go
type Memory interface {
    Add(role, content string)
    Messages() []ai.Message
    Clear()
}
```

`NewMemory(store, key, limit)` is the durable default; `NewInMemory(limit)` is non-persistent. `AgentTool` registers a function the model can call alongside the services it discovers.

Functional options:

```go
agent := micro.NewAgent("task-mgr",
    micro.AgentServices("task"),
    micro.AgentPrompt("You manage tasks. You understand deadlines and priorities."),
    micro.AgentProvider("anthropic"),
)
```

## Scoped Tools

An agent only sees the endpoints of its assigned services (plus excludes its own endpoints so it doesn't call itself).

## Memory

Agents persist conversation history in the store. Memory survives restarts.

```
agent/{name}/history    — conversation history
```

## Built-in Capabilities

Beyond its scoped service tools, every agent gets two built-in tools. They are not service endpoints — they are capabilities the agent has over itself and over other agents. They are plain tools wired into the agent's tool handler; there is no separate harness, loop engine, or graph. The LLM calls them exactly like any other tool.

### plan

For multi-step work the agent records an ordered plan: a list of steps, each with a `task` and a `status` (`pending`, `in_progress`, `done`). The plan is persisted to the store and surfaced back in the system prompt on later turns, so the agent stays oriented.

```
agent/{name}/plan       — current plan
```

### delegate

The agent hands a self-contained subtask to another agent. **Delegate-first** resolution:

1. If the target names a **registered agent** (a service advertising `type=agent`), the subtask is sent to it via RPC (`Agent.Chat`). Intelligence stays distributed — the domain expert handles its own services.
2. Otherwise a focused **ephemeral sub-agent** is created with `New(...)` + `Ask(...)`, given a fresh, isolated context, asked the subtask, and torn down.

A sub-agent is just an agent — no new "spawn"/"fork" concept. Ephemeral sub-agents load and persist no history and have no built-in tools, so they cannot plan or re-delegate (which bounds recursion).

These capabilities are added automatically to any non-ephemeral agent, so existing `NewAgent` services and `micro chat` routing get them for free.

## Registration

Agents register as real services via `server.NewServer` with metadata:

```go
server.Metadata(map[string]string{
    "type":     "agent",
    "services": "task,project",
})
```

The server has a real address, real transport, real endpoints. `micro agent list` discovers agents by checking server metadata for `type=agent`.

## The Router (micro chat)

`micro chat` is a router. It discovers agents from the registry and dispatches to them via RPC.

- One agent → routes directly via `client.Call(agentName, "Agent.Chat", ...)`
- Multiple agents → LLM classifies intent, calls `route_to_agent` tool
- No agents → falls back to direct service access (current behaviour)

## Agent-to-Agent Communication

Agents call each other via standard RPC. An agent is a service — it has an `Agent.Chat` endpoint. Any agent can call any other agent the same way it calls a service.

```go
// From inside an agent's logic, call another agent:
client.Call("comms-mgr", "Agent.Chat", &ChatRequest{Message: "Notify Alice"})
```

No special protocol. No broker topics. Just RPC.

## Usage Patterns

### Single-service agent

```go
agent := micro.NewAgent("task-mgr",
    micro.AgentServices("task"),
    micro.AgentPrompt("You manage tasks."),
    micro.AgentProvider("anthropic"),
)
agent.Run()
```

### Multi-service agent

```go
agent := micro.NewAgent("project-mgr",
    micro.AgentServices("task", "project", "milestone"),
    micro.AgentPrompt("You manage the project system."),
    micro.AgentProvider("anthropic"),
)
agent.Run()
```

### Programmatic

```go
agent := micro.NewAgent("support", ...)
agent.Init()
resp, _ := agent.Ask(ctx, "What tickets are open?")
```

### Agent alongside service

```go
func main() {
    svc := micro.New("task")
    svc.Handle(new(TaskHandler))

    agent := micro.NewAgent("task-mgr",
        micro.AgentServices("task"),
        micro.AgentPrompt("You manage tasks."),
        micro.AgentProvider("anthropic"),
    )

    go svc.Run()
    agent.Run()
}
```

## CLI

```bash
micro agent list                    # list registered agents
micro agent describe task-mgr       # show agent details
micro chat                          # routes to agents automatically
micro call task-mgr Agent.Chat '{"message": "..."}'  # direct RPC
```

## Generation

`micro run --prompt` creates services AND an agent:

```
micro run --prompt "task management system"

  Generated:
    task/       ← service
    project/    ← service
    agent/      ← agent (manages task, project)
```

The agent reads `MICRO_AI_PROVIDER` and `MICRO_AI_API_KEY` from the environment.

## What Doesn't Change

- Services are still services — same interface, same code, same deployment
- You can run services without agents
- You can call services directly via `micro call`, the API, or MCP
- The framework interfaces (registry, client, server, store) are unchanged
- `micro run`, `micro deploy`, `micro build` work the same way
