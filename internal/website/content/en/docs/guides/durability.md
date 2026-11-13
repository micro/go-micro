---
title: "Agent durability"
---

Agents can opt into checkpointing with `agent.WithCheckpoint` and
`agent.StoreCheckpoint`. Keep the store across restarts and recreate the agent
with the same name and configuration before resuming a run.

Completed tool results are reused on resume. A crash after an external action
but before its result is saved can still repeat that action. Make writes
idempotent; checkpointing does not provide exactly-once effects or a transaction
across services. Run one owner for each checkpointed execution.

Use `micro agent history` and `micro inspect agent` to inspect agent runs.
See the [durable agent example](https://github.com/micro/go-micro/tree/master/examples/agent-durable)
and [agent run records](agent-run-records.md).
