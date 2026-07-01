---
layout: default
---

# Learn by Example

Runnable examples are the fastest way to move from reading the guides to changing
one thing. Start with the path that matches where you are in the services →
agents → workflows lifecycle.

## Start here

| Goal | Runnable example | Why it is useful |
| --- | --- | --- |
| 0→1 service | [`examples/hello-world`](https://github.com/micro/go-micro/tree/master/examples/hello-world) | Smallest RPC service with a client call and health checks. |
| First service-backed agent | [`examples/agent-demo`](https://github.com/micro/go-micro/tree/master/examples/agent-demo) | Multi-service project/task/team app with agent playground integration. |
| 0→hero lifecycle | [`examples/support`](https://github.com/micro/go-micro/tree/master/examples/support) | No-secret support-desk story: typed services, an agent, an event-driven flow, and a guardrail. |
| Planning and delegation | [`examples/agent-plan-delegate`](https://github.com/micro/go-micro/tree/master/examples/agent-plan-delegate) | Two agents collaborate through `plan` and `delegate` over normal Go Micro RPC. |
| Durable workflows | [`examples/flow-durable`](https://github.com/micro/go-micro/tree/master/examples/flow-durable) | Ordered, checkpointed flow steps resume without duplicating completed side effects. |
| AI-callable services | [`examples/mcp`](https://github.com/micro/go-micro/tree/master/examples/mcp) | MCP examples that expose service endpoints as model tools. |

## Guide-to-example map

- [Getting Started](../getting-started.html) → run
  [`examples/support`](https://github.com/micro/go-micro/tree/master/examples/support)
  to see the full lifecycle before generating your own service.
- [Your First Agent](../guides/your-first-agent.html) → run
  [`examples/agent-demo`](https://github.com/micro/go-micro/tree/master/examples/agent-demo)
  or [`examples/support`](https://github.com/micro/go-micro/tree/master/examples/support)
  when you want a complete service-backed agent to inspect.
- [0→hero Reference](../guides/zero-to-hero.html) → run
  [`examples/support`](https://github.com/micro/go-micro/tree/master/examples/support)
  for the human-readable scenario, then `make harness` for the full CI contract.
- [Plan & Delegate](../guides/plan-delegate.html) → run
  [`examples/agent-plan-delegate`](https://github.com/micro/go-micro/tree/master/examples/agent-plan-delegate).
- [Agents and Workflows](../guides/agents-and-workflows.html) → run
  [`examples/flow-durable`](https://github.com/micro/go-micro/tree/master/examples/flow-durable)
  and [`examples/support`](https://github.com/micro/go-micro/tree/master/examples/support).

## Repository examples

See the repository [examples index](https://github.com/micro/go-micro/tree/master/examples)
for the complete runnable list, including deployment, auth, gRPC interop, MCP,
agent, and flow examples.

## More

- [Real-World Examples](realworld/index.md)
