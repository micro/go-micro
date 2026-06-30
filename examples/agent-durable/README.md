# Durable agent run resume

This example shows the agent-side counterpart to `examples/flow-durable`: an
agent run is checkpointed with the same `Checkpoint` interface used by flows,
then resumed after an interruption without repeating a completed side effect.
The sample uses an in-memory store to keep repeated local runs deterministic;
use your service store for process-restart recovery.

Run it with:

```sh
go run ./examples/agent-durable
```

The demo model calls `inventory.reserve`, then fails to mimic a process dying
after the tool call was checkpointed. `micro.AgentPending` finds the unfinished
run and `micro.AgentResume` continues it from the saved checkpoint. The final
`tool executions: 1` line is the important bit: the reservation tool was not
called a second time during resume.

In a service, use the same pattern at startup:

```go
pending, _ := micro.AgentPending(ctx, agent)
for _, run := range pending {
    _, _ = micro.AgentResume(ctx, agent, run.ID)
}
```

`context.Context` cancellation and deadlines are still honored by checkpoint
loads/saves, model calls, and tool calls. Runs with terminal statuses such as
`done`, `canceled`, and `expired` are not returned by `AgentPending`.
