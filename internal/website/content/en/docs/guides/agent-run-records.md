---
title: Agent run records
description: "Understand, inspect, and safely export the versioned execution record for a Go Micro agent run."
---

An agent run record is the durable operational history of one execution. It gives
operators and automation one stable object for answering: what ran, how it was
correlated, which model and tools were involved, where it checkpointed, and how
it finished.

List runs to find an id, then inspect the complete record:

```sh
micro inspect agent support
micro inspect agent support --run <run-id>
micro inspect agent support --run <run-id> --json
```

The human view prints a derived summary followed by the ordered event timeline.
The JSON view emits the public `agent.RunRecord` contract:

```json
{
  "schema_version": 1,
  "summary": {
    "run_id": "01J...",
    "agent": "support",
    "status": "done",
    "events": 4
  },
  "events": [
    {"time": "2026-09-06T12:00:00Z", "run_id": "01J...", "agent": "support", "kind": "run"},
    {"time": "2026-09-06T12:00:01Z", "run_id": "01J...", "agent": "support", "kind": "done"}
  ]
}
```

Use `schema_version` before decoding records across Go Micro releases. The
summary is derived from the events and can be rebuilt; the ordered event list is
the source of truth. Loading a complete record fails if a listed event cannot be
read or decoded, so callers never receive a silently truncated record. The older
`agent.LoadRunEvents` API retains its tolerant best-effort behavior.

## Lifecycle

A normal run begins with `run`, contains zero or more model, stream, tool,
checkpoint, and resume events, and ends with `done` or `error`. A process can
stop before a terminal event, so the derived status remains `running` until a
later resume or terminal event is recorded. A refusal can set the status to
`refused`; classified errors produce statuses such as `timeout`, `rate_limited`,
`auth`, `configuration`, `unavailable`, or `provider_error`.

Events are persisted under the agent's scoped state at
`agent/<name>/runs/<run-id>/...` and are ordered oldest first when loaded. The
record remains available after the process stops as long as the replacement
process uses the same agent name and state store.

## Fields

The summary contains run and parent ids, agent identity, trace/span correlation,
start and update times, duration, event count, derived status, latest checkpoint
and stage, latest event and error, and cumulative spend.

Each event contains its timestamp, run lineage, trace/span correlation, agent,
kind, and the metadata relevant to that kind. Model events can include provider,
model, attempt, latency, and token usage. Tool events can include tool name,
attempt, refusal, errors, and spend. Checkpoint events use `name` for the stage
and `status` for checkpoint state.

Go callers can load the same contract directly:

```go
record, err := agent.LoadRunRecord(stateStore, "support", runID)
```

## Privacy and redaction

Run records are metadata-first by default. Raw model replies, tool arguments,
and tool results are not part of the persisted timeline. Raw prompt input is not
stored unless `agent.TraceInputs(true)` is explicitly enabled; with that option,
the initial event's `name` field contains the prompt.

Treat records as potentially sensitive before exporting them:

- Error and refusal strings originate at provider or application boundaries and
  can contain user data or secrets. Redact them at those boundaries.
- Memory and checkpoints are stored separately and may contain full conversation
  or application state. A run record does not sanitize those stores.
- Run, parent, trace, and span ids reveal execution relationships even when
  payloads are absent.
- `agent.OnRunEvent(...)` receives events before persistence. Apply the same
  access controls and retention policy to any external sink.

Keep `TraceInputs` disabled unless the state store and every observability sink
are approved for prompt data. Prefer recording stable identifiers and counts
over copying payloads into event names or error strings.
