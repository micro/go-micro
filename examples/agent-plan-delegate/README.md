# Agent Plan & Delegate

Demonstrates the two built-in agent capabilities — **planning** and **delegation** — in a small multi-agent system.

## What it shows

| Capability | Tool | What happens |
|------------|------|--------------|
| **Planning** | `plan` | The coordinator records an ordered list of steps before doing multi-step work. The plan is saved to its store-backed memory and shown back to it on later turns. |
| **Delegation** | `delegate` | The coordinator hands the notification step to a separate `comms` agent. Because `comms` is a registered agent, the hand-off goes over RPC — not an in-process call. |

Both `plan` and `delegate` are added to every agent automatically. There's no harness or graph to configure: they're plain tools the model calls, the same as any service endpoint.

## Layout

```
task     (service)   Add, List          ← owned by coordinator
notify   (service)   Send               ← owned by comms
comms    (agent)     manages notify
coordinator (agent)  manages task, delegates notifications to comms
```

## Run

```bash
MICRO_AI_PROVIDER=anthropic MICRO_AI_API_KEY=sk-ant-... go run main.go
```

The coordinator is asked to *"Create three launch tasks: Design, Build, and Ship. Then make sure owner@acme.com is notified that the launch plan is ready."*

Expected shape of the run:

```
--- coordinator tool calls ---
  → plan({"steps":[{"task":"create Design task","status":"pending"}, ...]})
  → task_TaskService_Add({"title":"Design"})
  → task_TaskService_Add({"title":"Build"})
  → task_TaskService_Add({"title":"Ship"})
  → delegate({"task":"Notify owner@acme.com that the launch plan is ready","to":"comms"})
  📨 notify: to=owner@acme.com message="The launch plan is ready"

--- coordinator reply ---
Created the three launch tasks and asked comms to notify owner@acme.com.
```

## Delegate-first

`delegate` is hybrid:

1. If `to` names a **registered agent** that owns the relevant services, the subtask is sent to it over RPC (`Agent.Chat`). That's what happens here — `comms` owns `notify`.
2. Otherwise a focused **ephemeral sub-agent** is created for the subtask with a fresh, isolated context, asked the task, and torn down. Ephemeral sub-agents have no built-in tools, so they can't re-delegate.

This keeps intelligence distributed: the coordinator doesn't need to know how to send notifications — it knows *who does*.
