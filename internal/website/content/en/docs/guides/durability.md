---
title: "Durability and Recovery"
---

# Durability and recovery

Go Micro implements checkpointed flow steps and opt-in checkpointed agent runs.
Conversation memory, execution checkpoints, and tracing are separate mechanisms.
This page describes the shipped contract; the original
[design note](https://github.com/micro/go-micro/blob/master/internal/docs/DURABLE_EXECUTION_DESIGN.md)
is historical, not an implementation checklist.

## Storage and recovery boundaries

| Mechanism | Default and persisted state | Recovery boundary |
|---|---|---|
| Agent memory | Store-backed conversation history, unless custom or in-memory memory is selected | Restores conversation context; does not by itself resume interrupted tool execution |
| Ordered flow | Store-backed `flow.Checkpoint` when `Steps` is non-empty | Saves before and after each step; resumes from the saved `State.Stage` |
| Agent run | Opt in with `agent.WithCheckpoint` / `micro.AgentWithCheckpoint` | Saves run input, status, completed tool results, and final response for explicit resume |
| OpenTelemetry | Configure a trace provider | Observes runs and steps; spans are not the checkpoint store or a recovery scheduler |

`store.DefaultStore` is the file store under `~/micro/store/`. Its files must
survive the process/container lifetime; an ephemeral volume is not restart
recovery. `store.NewMemoryStore()` does not survive process exit. A custom store
or checkpoint backend determines its own persistence and failure guarantees.
`flow.StoreCheckpoint(store, scope)` isolates runs in database `flow`, table
`scope`; reuse the same backend and scope when recovering.

Recovery is explicit. Reconstruct the flow/agent with compatible configuration
and call the appropriate resume API. Persisting a checkpoint does not install a
background recovery worker or migrate checkpoints when step names/configuration
change.

## Flow runs, waiting, and retries

`flow.Run` holds the run ID and parent ID, flow name, state (`Stage` and serialized
`Data`), step records, status, optional awaited input, and timestamps. Successful
and failed runs are retained unless `DeleteOnSuccess` is enabled.

- `Resume(ctx, runID)` continues from the saved stage; a completed run is a no-op.
  A step whose completion was successfully saved is skipped. A step still
  recorded as `in_progress` can execute again.
- `Pending` excludes completed and `waiting` runs. `ResumePending` processes
  pending runs oldest first and stops at the first error, returning its run ID.
- `Await` / `AwaitStep` save a `waiting` run and return cleanly. `Waiting` lists
  those runs. `ResumeWith(ctx, runID, input)` uses the supplied string as the
  awaited step's output state and continues at the next step; it does not merge
  the string into the previous JSON payload automatically.
- `Retry(n)` allows up to `n` retries after the first attempt. A positive
  `Step.Retry` overrides it; `Step.Retry == 0` inherits the flow default, so zero
  does not disable an inherited retry policy. Context cancellation/deadlines
  stop retries, including backoff waits.
- `Loop` is one flow step. Its iterations are not independently checkpointed by
  the engine. `OnIteration` is a callback for application-owned progress; it does
  not automatically make the loop resume at that iteration.

A `Dispatch` step calls `Agent.Chat` and propagates the parent run ID. That
lineage is not an instruction to resume an existing child agent run: replaying
an interrupted dispatch can create another agent invocation.

## Agent runs and tool replay

With a checkpoint configured, `agent.Resume` returns a completed run's saved
response without invoking the model again. An unfinished resumable run re-enters
agent execution with its saved input and completed tool history; model work can
run again. `ResumeStreamAsk` supplies the streaming resume path. A human-input
pause requires `ResumeInput`, rather than plain `Resume`.

Completed tool results are reused within the run by **tool name plus
JSON-serialized input**, not by the provider's tool-call ID. Identical calls in
one run therefore share a completed result. This is not a cross-run idempotency
key and does not guarantee that a model will choose the same arguments after a
restart.

Agent pending runs exclude terminal `done`, `canceled`, `timeout`,
`rate_limited`, and `expired` statuses. Paused runs can still appear in pending
results; an input-required pause needs the input helper, so an unattended
`ResumePending` loop can stop there.

## What is not guaranteed

- **No exactly-once external side effects.** A crash after an external action
  succeeds but before its completion is saved can repeat that action. Flow
  retries can repeat it too. Use application/callee idempotency or reconciliation
  for payments, provisioning, and other irreversible actions.
- **No checkpoint transaction spanning an external service.** Flow checkpoint
  errors are returned, but the agent tool wrapper's intermediate saves are
  best-effort. Do not rely on replay suppression after a storage failure.
- **No multi-replica execution lease in the `Checkpoint` interface.** Its methods
  are Save, Load, Delete, and List; coordinate concurrent recovery externally.
- **No universal external-engine adapter.** `Checkpoint` is an extension seam;
  it does not itself provide a Temporal/Restate integration or their guarantees.

Checkpoint payloads and tool results can contain application data. Conversation
memory and checkpoints are not made private merely by enabling tracing; choose
appropriate storage access, retention, and redaction for each separately.

## No-secret verification

From the repository root, run the existing deterministic tests:

```sh
go test ./flow -run 'TestFlow(CheckpointResume|AwaitAndResumeWith|StepRetry|CheckpointSaveFailureStopsRun)'
go test ./agent -run 'Test(ResumeFailedCheckpointAfterFreshAgentRestart|ResumePendingAfterFreshAgentRestartDoesNotReplayCompletedTool|HumanInputPauseResumesSameRunWithInput|ResumeStreamAskDoesNotReplayCompletedTool)'
go test ./agent ./flow -run 'Test.*(OTel|Trace|Span)'
go run ./examples/agent-durable
```

These tests use controlled models/stores; the agent example uses an in-memory
store and demonstrates a checkpointed tool result being reused after a simulated
interruption. Fresh-agent tests reconstruct the agent over the same test store.
They are not process-kill, disk-loss, multi-replica, or real-provider guarantees.
For deployment validation, also exercise restart with the actual persistent
backend and volume configuration.

## Source of truth

- [Flow records, checkpoint backend, resume and retry](https://github.com/micro/go-micro/blob/master/flow/steps.go)
- [Loop boundary](https://github.com/micro/go-micro/blob/master/flow/loop.go)
- [Agent checkpoint and tool deduplication](https://github.com/micro/go-micro/blob/master/agent/checkpoint.go)
- [Agent execution and memory setup](https://github.com/micro/go-micro/blob/master/agent/agent.go)
- [Streaming resume](https://github.com/micro/go-micro/blob/master/agent/stream.go)
- [Default store](https://github.com/micro/go-micro/blob/master/store/store.go)
- [Remaining agent/flow design discussion](https://github.com/micro/go-micro/issues/4816)
