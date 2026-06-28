---
layout: default
---

# Agent2Agent (A2A)

Go Micro speaks the [Agent2Agent (A2A) protocol](https://a2a-protocol.org) — the open standard for agents on different frameworks to discover and call each other over HTTP. The A2A gateway is the agent-side analogue of the [MCP gateway](../mcp.html): MCP exposes your services as tools, A2A exposes your agents as agents.

There is nothing to add to an agent. An agent already registers in the registry with `type=agent` metadata; the gateway discovers it, generates an **Agent Card** from that metadata, and translates incoming A2A tasks to the agent's existing `Agent.Chat` RPC — the same call `delegate` and flows use.

## Run it

```bash
micro a2a serve --address :4000 --base_url https://agents.example.com
micro a2a list     # agents and their Agent Card URLs
```

Or embed the gateway next to a service:

```go
go a2a.Serve(a2a.Options{
    Registry: service.Options().Registry,
    Address:  ":4000",
    BaseURL:  "https://agents.example.com",
})
```

## Gateway, or directly on the agent

A2A is JSON-RPC over HTTP — a different wire protocol from go-micro's RPC — so *something* always translates between the two. That something doesn't have to be a separate process. There are two ways to run it:

- **A gateway** (above) fronts every agent in the registry behind one endpoint. Use it for a single front door, centralized discovery, and shared policy.
- **Directly on the agent.** `AgentA2A(addr)` makes the agent serve its own A2A endpoint when it runs — no separate gateway, and the task is handled in-process (no extra RPC hop):

  ```go
  agent := micro.NewAgent("task-mgr",
      micro.AgentServices("task"),
      micro.AgentProvider("anthropic"),
      micro.AgentA2A(":4000"),   // also reachable at http://host:4000 over A2A
  )
  agent.Run()
  ```

  The agent stays a normal go-micro service; this adds a second, A2A-native HTTP endpoint. Now any A2A client can `curl` it directly. Use it when each agent should be independently addressable without a gateway.

Both reuse the same handler; the only difference is whether the agent is reached over RPC (gateway) or in-process (embedded).

## Discovery: cards from the registry

Every registered agent gets an Agent Card, generated from its registry metadata (name, the services it manages). Cards are not published by the agent — they are derived, the same way MCP tools are derived from service endpoints.

| Endpoint | Returns |
|---|---|
| `GET /agents` | a directory of all Agent Cards |
| `GET /agents/{name}` | one agent's card |
| `GET /agents/{name}/.well-known/agent.json` | one agent's card (well-known path) |
| `POST /agents/{name}` | the agent's JSON-RPC endpoint |
| `GET /.well-known/agent.json` | the single agent's card, when exactly one is registered |

A card looks like:

```json
{
  "name": "task-mgr",
  "description": "Go Micro agent managing: task,project",
  "url": "https://agents.example.com/agents/task-mgr",
  "version": "1.0.0",
  "protocolVersion": "0.3.0",
  "capabilities": { "streaming": true, "pushNotifications": true },
  "defaultInputModes": ["text/plain"],
  "defaultOutputModes": ["text/plain"],
  "skills": [{ "id": "chat", "name": "Chat", "tags": ["task", "project"] }]
}
```

## Calling an agent

A2A uses JSON-RPC 2.0 over HTTP. Send a message with `message/send`; the gateway runs the agent and returns a completed `Task`:

```bash
curl -s https://agents.example.com/agents/task-mgr \
  -H 'content-type: application/json' \
  -d '{
    "jsonrpc": "2.0", "id": 1, "method": "message/send",
    "params": { "message": {
      "role": "user", "kind": "message", "messageId": "m1",
      "parts": [{ "kind": "text", "text": "What tasks are overdue?" }]
    }}
  }'
```

```json
{
  "jsonrpc": "2.0", "id": 1,
  "result": {
    "id": "…", "contextId": "…", "kind": "task",
    "status": { "state": "completed", "timestamp": "…" },
    "artifacts": [{ "artifactId": "…", "parts": [{ "kind": "text", "text": "Two: …" }] }]
  }
}
```

Retrieve a task later with `tasks/get` (`params: { "id": "…" }`). To continue
the same piece of work, send another `message/send` with the previous `taskId`
and `contextId`. The gateway preserves the task id, context id, and prior
history, then appends the new user turn and agent reply. That makes a remote
A2A task fit the Go Micro lifecycle: services are still invoked through the
agent's normal tools, the agent keeps task context across turns, and a workflow
can poll one task id as the conversation progresses.

## Push notifications

Operators can register a task callback with
`tasks/pushNotificationConfig/set`:

```bash
curl -s https://agents.example.com/agents/task-mgr \
  -H 'content-type: application/json' \
  -d '{
    "jsonrpc": "2.0", "id": 2,
    "method": "tasks/pushNotificationConfig/set",
    "params": {
      "id": "task-id",
      "pushNotificationConfig": {
        "url": "https://workflow.example.com/a2a/tasks",
        "token": "optional-bearer-token"
      }
    }
  }'
```

The gateway stores one callback per retained task and POSTs the latest task
snapshot to that URL whenever the task changes. Delivery is best effort: failures
do not fail the agent turn, and there is no retry queue in the in-memory gateway.
Use `tasks/get` as the source of truth after a missed callback or receiver
outage. If a token is configured, it is sent as `Authorization: Bearer <token>`.

## Calling out to other agents

The gateway makes your agents reachable *from* the A2A ecosystem. The
client (`a2a.Client`) is the other direction: it lets a Go Micro agent or
flow call an agent on any framework, by URL.

```go
reply, err := a2a.NewClient("https://other.example.com/agents/research").
    Send(ctx, "Summarize the latest on X")
```

It's wired into the two places that hand off work:

- **A flow step** — `flow.A2A(url)` is the cross-framework counterpart to
  `flow.Dispatch(name)` (which dispatches to a local agent):

  ```go
  flow.Step{Name: "research", Run: flow.A2A("https://other.example.com/agents/research")}
  ```

- **Agent delegate** — when an agent's `delegate` target is an `http(s)`
  URL, the subtask is sent to that external agent over A2A instead of to a
  locally registered one. Nothing else changes; the model just delegates
  to a URL.

`Send` handles the task lifecycle: if the remote returns a task that isn't
yet terminal, it polls `tasks/get` until it completes.

## Scope

This is the JSON-RPC binding for task execution:

- **`message/send`** runs the agent and returns a completed `Task`.
- **`message/stream`** streams the completed `Task` as an SSE `data:` event, giving A2A clients a streaming-compatible path while the underlying agent call remains synchronous.
- **`tasks/get`** returns a recent task by id.
- **Multi-turn continuation** keeps task state when a new message includes the previous `taskId`.
- **`tasks/pushNotificationConfig/set` / `get`** stores and reads a task callback for best-effort update delivery.
- **`tasks/resubscribe`** reconnects to an existing task stream, immediately emits the current task snapshot, then streams subsequent updates until the task reaches a terminal state.
- **`input-required`** task state carries human-input handoffs (for example checkpointed approval pauses) in task status, artifacts, and history; continue the task by sending a follow-up message with the same `taskId` and `contextId`.
- **Agent Card** discovery, generated from the registry.

Both directions work: the gateway exposes your agents, and `a2a.Client` (via `flow.A2A` or `delegate` to a URL) calls external ones. The task binding is what makes a Go Micro agent both reachable from, and able to reach, the A2A ecosystem today.

## See also

- [MCP & AI Agents](../mcp.html) — exposing services as tools
- [Agents and Workflows](agents-and-workflows.html) — the agent model
- [A2A protocol specification](https://a2a-protocol.org)
