# Agent Interface Design

## Principle

Service = capability. Agent = intelligence. They're separate. The service doesn't know about its agent. The agent is created to manage the service.

```
micro.New("task")           // creates a service
micro.NewAgent("task-mgr")  // creates an agent
```

Same package. Same level. Same patterns. Different responsibilities.

## Interface

```go
type Agent interface {
    Name() string
    Init(...AgentOption)
    Options() AgentOptions
    Chat(ctx context.Context, message string) (*AgentResponse, error)
    Run() error
    Stop() error
    String() string
}

type AgentResponse struct {
    Reply     string       // the agent's text response
    ToolCalls []ai.ToolCall // tools the agent called
    Agent     string       // which agent handled it
}
```

**Chat** is the core method. Send a message, get a response. The agent figures out which of its services to call, in what order, and returns a coherent answer.

**Run** registers the agent in the registry and blocks. The router and other agents can discover it.

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
    Client       client.Client     // call service endpoints
    Store        store.Store       // agent memory (persists across restarts)
    Broker       broker.Broker     // agent-to-agent communication
    HistoryLimit int               // max conversation turns to retain
}
```

Functional options follow the same pattern as Service:

```go
agent := micro.NewAgent("task-mgr",
    micro.AgentServices("task"),
    micro.AgentPrompt("You manage tasks. You understand deadlines and priorities."),
    micro.AgentProvider("anthropic"),
)
```

## Scoped Tools

An agent only sees the endpoints of its assigned services. A task agent doesn't see notification endpoints. This is different from today's `micro chat` which sees everything.

```go
// Inside the agent implementation:
tools := ai.NewTools(opts.Registry, ai.ToolClient(opts.Client))
discovered, _ := tools.Discover()

// Filter to only this agent's services
var scoped []ai.Tool
for _, t := range discovered {
    for _, svc := range opts.Services {
        if strings.HasPrefix(t.OriginalName, svc+".") {
            scoped = append(scoped, t)
        }
    }
}
```

## Memory

Agents persist conversation history and learned context in the store. Memory survives restarts.

```go
// On Chat(), before calling the LLM:
recs, _ := store.Read("agent/task-mgr/history")
// Deserialize into []ai.Message, append new message

// After response:
// Serialize updated history, write back
store.Write(&store.Record{Key: "agent/task-mgr/history", Value: data})
```

Memory is scoped per agent, per user (when auth is present):

```
agent/{name}/history              — conversation history
agent/{name}/context              — learned facts and preferences
agent/{name}/user/{id}/history    — per-user conversation history
```

The store backend determines durability. File store (default) persists locally. Postgres persists across machines.

## Registration

Agents register in the registry alongside services. Metadata distinguishes them:

```go
registry.Register(&registry.Service{
    Name: "task-mgr",
    Metadata: map[string]string{
        "type":     "agent",
        "services": "task",
    },
})
```

## The Router (micro chat)

`micro chat` is not an agent. It's a router. The single entry point that dispatches to agents.

When a message comes in, the router:
1. Discovers all agents from the registry
2. Uses an LLM to classify which agent(s) should handle the message
3. Calls `agent.Chat()` on the selected agent
4. Returns the response to the user

If a message spans multiple agents, the router coordinates:

```
> Reschedule Alice's tasks to next week and notify her

  Router → task-mgr:  "Reschedule Alice's tasks to next week"
  Router → comms-mgr: "Notify Alice her tasks were rescheduled"
```

The router is lightweight — it doesn't have domain knowledge. It reads agent descriptions from the registry and routes based on intent. Like Claude spawning sub-agents, you talk to one interface and it delegates to specialists.

If no agents are registered, the router falls back to the current behaviour — discovers all services, sees all endpoints, acts as a single general-purpose agent. This preserves backward compatibility.

```bash
# Full system: router dispatches to agents
micro chat
> What tasks are overdue?
  [task-mgr] You have 3 overdue tasks...

# No agents registered: fallback to direct service access
micro chat
> What tasks are overdue?
  → task_Task_ListOverdue(...)
```

## Hot Reload

Agents can be reloaded without restart when their prompt or service assignments change.

The `micro run` watcher detects changes to agent files the same way it detects changes to service files. When an agent's source changes, it rebuilds and restarts the agent binary.

For prompt-only changes (no code change), the agent watches its prompt source (file, config, or store) and reinitializes its LLM context without restarting.

```go
// Agent watches its own prompt key in the config/store
// and re-initializes when it changes:
go func() {
    watcher, _ := config.Watch("agent", "task-mgr", "prompt")
    for {
        v, _ := watcher.Next()
        agent.updatePrompt(v.String(""))
    }
}()
```

## Usage Patterns

### Single-service agent

```go
agent := micro.NewAgent("task-mgr",
    micro.AgentServices("task"),
    micro.AgentPrompt("You manage tasks. You understand deadlines, priorities, and assignments."),
    micro.AgentProvider("anthropic"),
)
agent.Run()
```

### Multi-service agent

```go
agent := micro.NewAgent("project-mgr",
    micro.AgentServices("task", "project", "milestone"),
    micro.AgentPrompt("You manage the project system. Tasks belong to projects. Milestones track progress."),
    micro.AgentProvider("anthropic"),
)
agent.Run()
```

### Programmatic chat

```go
agent := micro.NewAgent("support", ...)
agent.Init()
resp, _ := agent.Chat(ctx, "What tickets are open for Alice?")
fmt.Println(resp.Reply)
```

### Agent alongside service in the same binary

```go
func main() {
    svc := micro.New("task")
    svc.Handle(new(TaskHandler))

    agent := micro.NewAgent("task-mgr",
        micro.AgentServices("task"),
        micro.AgentPrompt("You manage the task service."),
        micro.AgentProvider("anthropic"),
    )

    // Run both
    g := micro.NewGroup(svc)
    go g.Run()
    agent.Run()
}
```

## CLI

```bash
# List agents
micro agent list
  task-mgr       manages: task
  project-mgr    manages: task, project, milestone

# Chat with a specific agent directly
micro agent chat task-mgr
> What tasks are overdue?

# micro chat routes automatically
micro chat
> What tasks are overdue?
  [task-mgr] You have 3 overdue tasks...

# micro chat with no agents falls back to current behaviour
micro chat
> What tasks are overdue?
  → task_Task_ListOverdue(...)
```

## Agent-to-Agent Communication

Agents talk through the broker, not by calling each other's services:

```
> Reschedule Alice's tasks and notify her

  [task-mgr] Rescheduling 3 tasks...
  [task-mgr] Publishing to agent.comms-mgr: "Alice's tasks rescheduled"
  [comms-mgr] Sending notification to Alice...
```

The broker topic convention: `agent.{name}` for direct messages, `agent.broadcast` for announcements.

## Generation

`micro run --prompt` creates services AND their agent:

```
micro run --prompt "task management system"

  Generated:
    task/           ← service
    project/        ← service
    task-mgr/       ← agent (manages task, project)
```

The agent is a separate binary with its own `main.go`:

```go
package main

import "go-micro.dev/v5"

func main() {
    agent := micro.NewAgent("task-mgr",
        micro.AgentServices("task", "project"),
        micro.AgentPrompt("You manage tasks and projects..."),
        micro.AgentProvider("anthropic"),
    )
    agent.Run()
}
```

## What Doesn't Change

- Services are still services — same interface, same code, same deployment
- You can run services without agents
- You can call services directly via `micro call`, the API, or MCP
- The framework interfaces (registry, broker, store, client, server) are unchanged
- `micro run`, `micro deploy`, `micro build` work the same way
