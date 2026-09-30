---
title: "Execution and recovery contract"
description: "How services, agents, and flows compose, retry, and recover."
---

Go Micro coordinates work through services, agents, and flows. Services own
capabilities and their effects; agents own adaptive tool execution; flows own
explicit step order. The implementation is additive in v6 so applications can
migrate before a major-version API removal.

## Execution matrix

| Entry point | Execution owner | Retry boundary | Recovery |
|-------------|-----------------|----------------|----------|
| Direct service RPC | Service/client | Client configuration | Service owns idempotency and reconciliation |
| Agent Ask / Chat | Agent harness | One provider turn for built-in providers | Configured checkpoint; stable local `agent.Run` for deduplication |
| Agent StreamAsk / StreamChat | Same harness | Same as Ask | Tool events and completed-answer chunks; resume uses saved run |
| Native model/agent Stream | Provider text stream | Caller | No durable tool execution; use StreamAsk for work |
| Prompt flow / LLM step | Local agent harness | Harness model policy within flow step policy | Strict stepped runs use a stable child agent ID |
| Ordered flow | Flow engine | Named step | Checkpoint between steps; strict mode persists attempt admission |
| Broker flow | Flow engine | Broker redelivery and step policy | Strict mode requires `micro-run-id`; acknowledgment follows recorded completion/waiting |

A successful reply is not proof of a business effect. Services enforce their
invariants; step verifiers can reject unsuitable outputs. Waiting for input is
not completion. Cancellation and exhaustion are terminal in strict flow recovery.
Timeouts and external failures can leave an ambiguous outcome.

## Shared harness and provider turns

Built-in providers implement `model.Turner`: one request returns text or
structured tool calls, usage, stop reason, and opaque continuation. It never
executes tools. The agent harness performs tool calls, applies guardrails, then
returns the ordered results with that continuation. Signed thinking and other
provider fields survive round trips. Provider retries apply to that model turn,
not to the already executed tools preceding it.

`Generate` retains its legacy v6 behavior for direct callers. Third-party model
wrappers must preserve the optional Turner interface to use the new path.
Strict agent recovery rejects providers without it. Native provider token
streaming is distinct from the harness's tool-event/answer-chunk stream.

Flow `LLM` and prompt execution now use the same harness as agents and CLI chat.
Use `flow.AgentOptions` for explicit service scope and agent limits. Each local
invocation has its own memory/store by default; hosts configure persistent
conversation or plan storage separately. A plain model call remains available
inside a custom step with no tool handler.

## Strict recovery

Use `flow.StrictRecovery()` with a fenced checkpoint. `flow.OpenCheckpoint(path)`
opens a persistent local journal with an exclusive process lock and per-run
execution locks. It supports one host process, not a replicated cluster.
Distributed backends must implement the checkpoint and fenced ownership contract;
a generic store without atomic ownership is rejected in strict mode.

`Flow.Start(ctx, stableID, input)` deduplicates admission. Hosts must supply stable,
unique IDs and resume work after restart. Broker messages use `micro-run-id`.
A broker must actually support redelivery for handler errors to cause retries;
Go Micro cannot add that property to an in-memory broker.

Strict flows save attempt consumption before each step executes. A crash can
consume an attempt without producing an effect; recovery does not replenish the
budget. `Step.Idempotent` explicitly permits replay of an interrupted step.
Without it, an in-progress operation is ambiguous and stops for reconciliation.
A changed step list or unknown schema is rejected rather than silently restarted.
Keep step semantics stable while old runs exist, or migrate them explicitly.

Service steps attach a stable `micro-idempotency-key`. The receiving service must
atomically store that key with its effect, or use an external system's equivalent.
An action succeeding just before a checkpoint failure remains ambiguous without
that cooperation. Persisted state alone does not provide exactly-once effects.

Strict local agents persist pending provider turns, continuation, tool results,
and consumed model/tool budgets. Recovered completed calls are reused; an
interrupted unrecorded tool effect stops for reconciliation. Checkpoint failures
are surfaced rather than reported as successful execution. Human input starts a
new model continuation while retaining completed tool results and budgets.

`Loop` reports `flow.ErrLimit` when a configured condition is not met by its cap.
Strict mode rejects whole-loop replay; expand durable loops into named steps.
Remote Dispatch requires every registry record to advertise `stable_runs=v1`.
Strict checkpointed agents advertise this capability and accept a stable child
run ID via RPC metadata; older agents are rejected instead of silently replayed.
Use explicit non-strict behavior only when its weaker recovery contract is suitable.

## Acceptance and migration

Run the [durable-flow example](https://github.com/micro/go-micro/tree/master/examples/durable-flow)
through start, exit, inspect, approve, and redelivery. It combines actual local
service RPC, a mock-model agent, an approval boundary, and persistent effects.
No provider subscription or Mu installation is needed.

Before adopting these changes:

1. Give adaptive flows an explicit service scope and agent limits. Built-in plan
   and input tools are now available through the shared harness.
2. Handle exhaustion separately from successful loop completion. Broker failures
   now return errors; audit your broker's retry and dead-letter configuration.
3. Configure persistent checkpoints and host ownership before enabling strict
   recovery. Mark a step idempotent only when its implementation supports replay.
4. Preserve `Turner` in provider wrappers; direct v6 `Generate` callers continue
   to work. A future major release can remove provider-owned tool execution once
   plugins and callers have migrated.
5. Version stored execution data. Current readers accept the legacy unversioned
   run shape and version 1; unsupported versions fail explicitly.

The remaining v7 release work is removal of the legacy provider-owned loops and
module-version migration after users adopt the one-turn contract. Distributed
checkpoint backends and per-iteration loop journaling are optional extensions;
the local journal and named-step recovery contract do not claim those features.
