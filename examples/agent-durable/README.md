# Durable agent run resume

This example shows an agent run checkpointed through `agent.Checkpoint`,
so interrupted work can resume without repeating recorded tool results.

Run it with:

```sh
go run ./examples/agent-durable
```

The demo model calls `inventory.reserve`, then fails to mimic a process dying
after the tool call was checkpointed. `micro.AgentPending` finds the unfinished
run and `micro.AgentResume` continues it from the saved checkpoint. The final
`tool executions: 1` line is the important bit: the reservation tool was not
called a second time during resume.
