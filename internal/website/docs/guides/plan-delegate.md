---
layout: default
---

# Plan & Delegate

Every Go Micro agent has two built-in capabilities, on top of the service tools it discovers:

- **`plan`** — record an ordered plan in memory before doing multi-step work.
- **`delegate`** — hand a self-contained subtask to another agent.

They are exposed to the model as ordinary tools. There is no separate graph runtime to configure — these harness capabilities are tools, and the agent calls them the same way it calls a service endpoint. They are added automatically to every agent, so you don't wire anything up. `micro chat` exposes them too, so you get planning and delegation even when talking to your services directly.

## Prerequisites

- Go 1.21+
- An API key for any supported provider (Anthropic, OpenAI, Gemini, Groq, Mistral, Together, Atlas Cloud)

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

## Smallest possible agent

An agent doesn't need any services to plan — `plan` and `delegate` are always available.

```go
package main

import (
	"context"
	"fmt"
	"os"

	"go-micro.dev/v6"
)

func main() {
	a := micro.NewAgent("assistant",
		micro.AgentProvider("anthropic"),
		micro.AgentAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
	)

	resp, err := a.Ask(context.Background(),
		"Plan how to launch a product, then carry out what you can.")
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Reply)
}
```

Save it in a fresh module and run:

```bash
mkdir my-agent && cd my-agent
go mod init my-agent
go get go-micro.dev/v6
# save the code above as main.go
export ANTHROPIC_API_KEY=sk-ant-...
go run main.go
```

The agent records its plan with the `plan` tool, then works through it. The plan is saved to the agent's store-backed memory and shown back to it on later turns, so it stays oriented across a long task.

## plan

The model calls `plan` with an ordered list of steps, each with a `task` and a `status` (`pending`, `in_progress`, `done`):

```json
{
  "steps": [
    {"task": "draft the announcement", "status": "in_progress"},
    {"task": "schedule the email",     "status": "pending"},
    {"task": "publish the blog post",  "status": "pending"}
  ]
}
```

The plan is persisted under `agent/{name}/plan` in the [store](../store.html) — file-backed by default, Postgres or NATS KV in production — and re-injected into the system prompt on subsequent turns. Memory survives restarts.

You don't have to do anything to enable this. Nudge the agent to use it from the prompt when you want disciplined multi-step behaviour:

```go
micro.AgentPrompt("For multi-step requests, call the plan tool first to record your steps, then carry them out.")
```

## delegate

`delegate` hands a self-contained subtask to another agent. It resolves **delegate-first**:

1. **If `to` names a registered agent** that owns the relevant services, the subtask is sent to it over RPC (`Agent.Chat`). The domain expert handles its own services.
2. **Otherwise** a focused, short-lived **sub-agent** is created for the subtask with a fresh, isolated context, asked the task, and torn down.

A sub-agent is just an agent — created with `New`, talked to with `Ask`. There is no separate "spawn" or "fork" concept to learn. Ephemeral sub-agents load and persist no history and have no built-in tools, so they can't plan or re-delegate — which keeps delegation from recursing.

```json
{
  "task": "Notify owner@acme.com that the launch plan is ready",
  "to": "comms"
}
```

This is how intelligence stays distributed: an agent doesn't need to know *how* to do everything, only *who* does. It mirrors how Go Micro already works — agents are services, and services call each other over RPC.

## A multi-agent example

Two services (`task`, `notify`) and two agents. The `conductor` owns `task`; `comms` owns `notify`. Asked to create tasks and notify someone, the conductor plans the work, creates the tasks with its own tools, then delegates the notification to `comms` — which, being a registered agent, receives the hand-off over RPC.

```go
comms := micro.NewAgent("comms",
	micro.AgentServices("notify"),
	micro.AgentPrompt("You handle outbound notifications."),
	micro.AgentProvider("anthropic"),
	micro.AgentAPIKey(key),
)
go comms.Run()

conductor := micro.NewAgent("conductor",
	micro.AgentServices("task"),
	micro.AgentPrompt(
		"For multi-step requests, call the plan tool first. "+
			"For notifications, delegate to the \"comms\" agent (to: \"comms\")."),
	micro.AgentProvider("anthropic"),
	micro.AgentAPIKey(key),
)

resp, _ := conductor.Ask(ctx,
	"Create three launch tasks: Design, Build, and Ship. "+
		"Then make sure owner@acme.com is notified that the launch plan is ready.")
```

A typical run:

```
→ plan({"steps":[{"task":"create Design task","status":"pending"}, ...]})
→ task_TaskService_Add({"title":"Design"})
→ task_TaskService_Add({"title":"Build"})
→ task_TaskService_Add({"title":"Ship"})
→ delegate({"task":"Notify owner@acme.com that the launch plan is ready","to":"comms"})
  📨 notify: to=owner@acme.com message="The launch plan is ready"
```

The full, runnable code is in [examples/agent-plan-delegate](https://github.com/micro/go-micro/tree/master/examples/agent-plan-delegate).

## When to use what

| You want… | Use |
|-----------|-----|
| The agent to stay on track over a long, multi-step task | `plan` |
| One domain expert to handle its own services | `delegate` with `to` set to that agent |
| A focused helper for a one-off subtask, with its own clean context | `delegate` with no matching agent (ephemeral sub-agent) |

## How it fits

`plan` and `delegate` don't add a new layer to the framework — they're tools, the same primitive everything else uses. That's deliberate: services are the only abstraction, the LLM calls them as tools, and an agent's own capabilities are no exception.

- [Agent Integration Patterns](agent-patterns.html) — Pattern 9 covers planning and delegation
- [AI Integration](../ai-integration.html) — agents, flows, and the model interface
- [Store](../store.html) — where agent memory lives
