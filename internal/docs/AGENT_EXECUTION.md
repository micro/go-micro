# Agent execution

Go Micro exposes services and agents. An agent uses a model to choose calls to
service tools. Its execution machinery belongs to the agent, just as request
handling belongs to a service's server.

## Responsibilities

- `Agent` configures and hosts an agent: `Ask`, `Stream`, `Run`, `Stop`.
- `TaskAgent` provides background work: `StartTask`, `GetTask`, `StopTask`.
- `Checkpoint` persists progress: `Save`, `Load`, `List`, `Delete`.
- `model.Model` talks to a provider; it does not own host lifecycle or schedules.
- `client.Client` calls services; `server.Server` exposes the agent over RPC.

A checkpoint is storage, not an executor. The default agent implementation owns
the model/tool loop, cancellation, approvals, and recovery. A new execution
interface should only be introduced when there is a concrete alternative
implementation to substitute; it should not duplicate `Agent` or `TaskAgent`.

## One path for agent work

Direct requests execute through the agent. Background tasks and schedules use
the same agent execution code. Scheduled tasks have their own task/run ID,
with schedule origin recorded as metadata; no enclosing workflow is needed.
Tasks survive a disconnected caller, not a stopped host. Schedules reserve the
next interval before dispatch and do not automatically replay interrupted work.
Checkpoint recovery is explicit because a tool may have performed an external
action before its result was saved.

`agent/run` owns persisted record types and the store implementation. The public
agent API exposes `Run`, `Checkpoint` and `StoreCheckpoint`, so applications can
configure recovery without importing a workflow engine.

## v6 compatibility and v7 boundary

The existing `flow` types alias the same records and checkpoint interface.
Store scope and JSON fields remain unchanged, including the historical `flow`
field. Existing checkpoints and custom implementations remain usable.

Ordered workflows remain an optional v6 API for predefined sequences of service
calls and agent tasks. They are not required for agent execution. The existing
prompt/provider flow API remains for compatibility; v7 should retire that second
model execution path in favor of dispatching to an agent. Event subscriptions
should invoke the agent through its normal request/task path.

Mu owns accounts, conversations, rendered replies and application scheduling.
Its saved conversation turn is not a workflow definition. It can link that turn
to an agent run ID without adopting the framework's record as its UI data model.
