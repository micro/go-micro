---
title: Flow run records
description: "Inspect a versioned Go Micro flow execution and navigate its service and agent lineage."
---

A flow run record is the durable map of one workflow execution: how it started,
which steps ran, which services they called, and which agent runs they created.
List runs to find an id, then inspect the complete record:

```sh
micro inspect flow daily-ops
micro inspect flow daily-ops --run <run-id>
micro inspect flow daily-ops --run <run-id> --json
```

The human view prints dispatch and trigger origin followed by every step. A
`flow.Call` step records its service and endpoint. A `flow.Dispatch` step records
the agent name and returned child run id. For each child it prints a command you
can follow directly:

```sh
micro inspect agent <agent> --run <child-run-id>
```

The JSON view emits the public `flow.RunRecord` contract:

```json
{
  "schema_version": 1,
  "run": {
    "id": "flow-01J...",
    "flow": "daily-ops",
    "dispatch": "schedule",
    "trigger": "daily-review",
    "status": "done",
    "steps": [
      {"name": "load", "status": "done", "service": "reports", "endpoint": "Reports.Load"},
      {"name": "summarize", "status": "done", "agent": "ops", "child_run_id": "agent-01J..."}
    ]
  }
}
```

Use `schema_version` before decoding records across Go Micro releases. Go
callers can load the same contract from any checkpoint implementation:

```go
record, err := flow.LoadRunRecord(ctx, checkpoint, "daily-ops", runID)
```

Execution identity also travels in Go Micro RPC metadata. A service or agent can
call `ai.RunInfoFrom(ctx)` to recover the flow run, current step, dispatch, and
trigger without changing its request schema. Agent runs replace the run id with
their own identity and retain the flow run as `parent_id`; their service tools
receive that same lineage.

Unlike the agent event timeline, a flow checkpoint includes `state.data` and
truncated step results so it can resume work. Treat the whole record as
application data. Names, ids, errors, triggers, and endpoint names can also be
sensitive. Apply the same access controls and retention policy used for flow
checkpoints before exporting a record.
