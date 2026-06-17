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

Or embed it next to a service:

```go
go a2a.Serve(a2a.Options{
    Registry: service.Options().Registry,
    Address:  ":4000",
    BaseURL:  "https://agents.example.com",
})
```

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
  "capabilities": { "streaming": false, "pushNotifications": false },
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

Retrieve a task later with `tasks/get` (`params: { "id": "…" }`).

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

This is the synchronous JSON-RPC binding:

- **`message/send`** runs the agent and returns a completed `Task`.
- **`tasks/get`** returns a recent task by id.
- **Agent Card** discovery, generated from the registry.

Both directions work: the gateway exposes your agents, and `a2a.Client` (via `flow.A2A` or `delegate` to a URL) calls external ones.

Not yet supported (advertised as such on the card, so clients negotiate correctly):

- **`message/stream`** (SSE streaming) and `tasks/resubscribe`.
- Multi-turn `input-required` tasks.
- Push notifications.

These are the natural follow-ups; the synchronous binding is what makes a Go Micro agent both reachable from, and able to reach, the A2A ecosystem today.

## See also

- [MCP & AI Agents](../mcp.html) — exposing services as tools
- [Agents and Workflows](agents-and-workflows.html) — the agent model
- [A2A protocol specification](https://a2a-protocol.org)
