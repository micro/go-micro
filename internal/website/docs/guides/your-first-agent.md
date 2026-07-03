---
layout: default
---

# Your First Agent

This walkthrough builds the smallest useful Go Micro agent path: one service
with typed endpoints, one agent scoped to that service, and one CLI conversation
that proves the agent can use the service as a tool. It is the 0→1 version of
the services → agents → workflows lifecycle: build capability first, add
intelligence on top, then keep a clear path toward flows when the work needs to
run on events or schedules.

## Runnable reference first

If you want to run the lifecycle before copying code, start with the [no-secret first-agent transcript](no-secret-first-agent.html) or run the maintained support-desk example from the repository root:

```sh
go run ./examples/support
```

It uses a deterministic mock model by default, so it needs no provider key, and it exercises the same shape this guide teaches: services become tools, an agent uses them, and a flow can trigger the work. Use the transcript for expected output, then use this guide when you are ready to build the smaller 0→1 version yourself.

## What you'll build

A tiny task assistant:

1. A `task` service exposes `Create` and `List` endpoints.
2. An `assistant` agent is scoped to the `task` service.
3. `micro run` starts both in the local harness.
4. `micro chat` asks the agent to create and list tasks.

The same service endpoints are normal RPC methods, dashboard/API actions, MCP
tools, and agent tools. You do not write a second integration layer for the
agent.

## Prerequisites

- Go 1.24 or newer.
- The `micro` CLI installed.
- An LLM provider key for live agent calls. For example:

```sh
export ANTHROPIC_API_KEY=sk-ant-...
```

Plain service calls work without a model key; the key is only needed when the
agent reasons over tools.

Run the read-only first-agent preflight before starting the walkthrough. The same CLI boundary is covered by CI with `go test ./cmd/micro -run TestFirstAgentWalkthroughCLIBoundaries -count=1`, so the documented scaffold → run → chat → inspect path stays visible in the local harness:

```sh
micro agent preflight
```

It checks Go 1.24+, the `micro` binary, provider-key setup, and the default local gateway port without contacting a provider. Failed checks include a `Fix:` line and a `Next:` line that points back to this guide, the no-secret walkthrough, or the debugging guide.

## 1. Create a workspace

```sh
mkdir first-agent
cd first-agent
go mod init example.com/first-agent
go get go-micro.dev/v6@v6
```

Add `main.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	micro "go-micro.dev/v6"
)

type CreateRequest struct {
	Title string `json:"title"`
}

type CreateResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type ListRequest struct{}

type ListResponse struct {
	Tasks []CreateResponse `json:"tasks"`
}

type TaskService struct {
	mu    sync.Mutex
	next  int
	tasks []CreateResponse
}

// Create adds a task to the list.
// @example {"title":"Write first agent guide"}
func (t *TaskService) Create(ctx context.Context, req *CreateRequest, rsp *CreateResponse) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.next++
	*rsp = CreateResponse{ID: fmt.Sprintf("task-%d", t.next), Title: req.Title}
	t.tasks = append(t.tasks, *rsp)
	return nil
}

// List returns all known tasks.
// @example {}
func (t *TaskService) List(ctx context.Context, req *ListRequest, rsp *ListResponse) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	rsp.Tasks = append([]CreateResponse(nil), t.tasks...)
	return nil
}

func main() {
	service := micro.NewService("task")
	service.Handle(new(TaskService))

	agent := micro.NewAgent("assistant",
		micro.AgentServices("task"),
		micro.AgentPrompt("You help manage tasks. Use the task service before answering."),
		micro.AgentProvider("anthropic"),
		micro.AgentAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
	)

	go agent.Run()
	service.Run()
}
```

> Why the comments matter: endpoint comments and `@example` tags become tool
> descriptions, so the agent has enough context to choose `task.Create` and
> `task.List` correctly.

## 2. Run the service and agent

From the same directory:

```sh
micro run
```

The local harness starts the service, gateway, dashboard, MCP tool surface, and
agent playground. You can also verify the service directly before involving the
agent:

```sh
micro call task TaskService.Create '{"title":"Ship the walkthrough"}'
micro call task TaskService.List '{}'
```

## 3. Chat with the agent

In another terminal, ask the agent to use the service:

```sh
micro chat assistant
```

Try:

```text
Create a task called "Review the first-agent walkthrough", then show me all tasks.
```

A healthy run shows the agent calling the task service and then summarizing the
result. If the model refuses to call tools, tighten the prompt so it explicitly
uses the `task` service before answering.

## 4. Know what just happened

- The service registered typed RPC endpoints.
- Go Micro derived tool descriptions from the endpoint names, comments, request
  fields, and examples.
- The agent registered as another service with an `Agent.Chat` endpoint.
- `micro chat` sent your message to the agent.
- The agent selected the scoped `task` tools, called them over the same runtime,
  and stored conversation history in memory.

That is the core lifecycle: services provide capability, agents use the
capability, and the same runtime can later put the interaction behind a flow.

## 5. Make it a workflow when the path is event-driven

Once the prompt should run because something happened rather than because a
human typed a message, move the handoff into a flow:

```go
flow := micro.NewFlow("task-triage",
	micro.FlowTrigger("tasks.created"),
	micro.FlowPrompt("Review this new task and decide the next action: {{.Data}}"),
	micro.FlowProvider("anthropic"),
	micro.FlowAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
)
```

Use flows for deterministic triggers and long-running orchestration; keep the
agent for judgment, tool use, and handoffs when the path is not known up front.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| The agent says it cannot access tasks. | Confirm the agent was created with `micro.AgentServices("task")` and that `micro agent list` shows `assistant`. |
| Tool calls use the wrong fields. | Add or improve doc comments and `@example` tags on the service methods. |
| Plain service calls work but chat fails. | Check that your provider key is exported in the shell that runs `micro run`. |
| You need a no-secret reference path. | Run `make harness` from the Go Micro repository; it exercises the services → agents → workflows lifecycle with a mock provider. |

## Next steps

- Read the [0→hero reference path](zero-to-hero.html) for the CI-verified
  lifecycle contract.
- Run the [no-secret first-agent transcript](no-secret-first-agent.html) or [`examples/support`](https://github.com/micro/go-micro/tree/master/examples/support) for the no-secret support-desk lifecycle.
- Run [`examples/agent-plan-delegate`](https://github.com/micro/go-micro/tree/master/examples/agent-plan-delegate)
  to see planning and delegation across agents.
- Read [Debugging your agent](debugging-agents.html) when a chat turn does not call the tool you expected, loops, refuses a call, loses memory, or fails after a flow handoff.
- Read [Agents and Workflows](agents-and-workflows.html) when you are ready to
  compose agents behind durable flows.
