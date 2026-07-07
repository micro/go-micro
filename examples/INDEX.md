# Examples wayfinding

Use this index when you want the shortest path from a first runnable agent to the
next services, agents, workflows, and interop examples. Every command below is
provider-free unless the example README says otherwise.

## Pick by goal

| Goal | Start here | Run or verify | Then try |
|------|------------|---------------|----------|
| Run the smallest no-secret agent | [`first-agent`](./first-agent/) | `go run ./examples/first-agent` | [`agent-demo`](./agent-demo/) for a larger service-backed agent |
| Prove the maintained 0→hero path | [`support`](./support/) | `go run ./examples/support` and `go test ./examples/support` | [`zero-to-hero` guide](../internal/website/docs/guides/zero-to-hero.md) |
| See planning and delegation | [`agent-plan-delegate`](./agent-plan-delegate/) | `go run ./examples/agent-plan-delegate` | [`plan-delegate` guide](../internal/website/docs/guides/plan-delegate.md) |
| Expose services through MCP | [`mcp/hello`](./mcp/hello/) | follow [`mcp`](./mcp/) setup | [`mcp/crud`](./mcp/crud/) and [`mcp/workflow`](./mcp/workflow/) |
| Try A2A or gRPC interop next | [`agent-demo`](./agent-demo/) plus gateway docs | run the example, then use the gateway docs | [`grpc-interop`](./grpc-interop/) |
| Add workflow durability | [`flow-durable`](./flow-durable/) | `go run ./examples/flow-durable` | [`flow-loop`](./flow-loop/) |

## Recommended adoption path

1. **First service:** run [`hello-world`](./hello-world/) to learn service
   registration, handlers, client calls, and health checks.
2. **First agent:** run [`first-agent`](./first-agent/) with
   `go run ./examples/first-agent`; it uses a deterministic mock model and needs
   no provider key.
3. **0→hero reference:** run [`support`](./support/) with
   `go run ./examples/support`; it keeps typed services, an agent chat loop, an
   event-driven flow, and an approval gate in one maintained example.
4. **Interop next:** use [`mcp/hello`](./mcp/hello/), [`mcp/crud`](./mcp/crud/),
   and [`mcp/workflow`](./mcp/workflow/) when you are ready to expose tools to
   external AI clients.
5. **Workflow depth:** use [`flow-durable`](./flow-durable/) once the agent path
   needs checkpointed, resumable deterministic work.

## CLI wayfinding

The installed CLI prints the same path:

```bash
micro examples
micro agent demo
micro zero-to-hero
```

Keep this file, [`README.md`](../README.md), and the `micro examples` output in
sync so new developers can find `examples/first-agent` and `examples/support`
from one documented path.
