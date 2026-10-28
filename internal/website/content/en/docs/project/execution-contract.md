---
title: "Execution Contract"
description: "The audited v6 execution boundaries and v7 compatibility decisions."
---

# Execution contract

This is the Stage 1 contract matrix tracked by
[#5044](https://github.com/micro/go-micro/issues/5044). It describes v6 as it
exists; it does not promise exactly-once effects or pre-approve a v7 API. An
invocation succeeding means its local boundary completed, not that an external
business effect is proven.

## Boundary matrix

| Boundary | Execution and state owner | Retry owner | Completion and cancellation | Persistence and recovery | Known gap / decision |
| --- | --- | --- | --- | --- | --- |
| Direct service RPC | Client and service handler | Caller/client wrappers | RPC response or error; the request context carries cancellation | None in the core RPC contract | Effects require an application idempotency key or reconciliation. No v7 break is required. |
| Direct agent `Ask` | Agent harness | Agent/model provider according to their configured limits | A response completes the run; errors are classified; context cancellation cancels active work | Opt-in checkpoint records run, tool results, and stage; `Resume` is explicit | Providers may still own tool iteration. Moving all iteration to the harness is a v7 break. |
| Streaming agent request | Agent harness and stream producer | Same as direct agent execution | Terminal event/error closes the stream; context or disconnected consumer cancels active delivery | Checkpoint and `ResumeStreamAsk` use the same run identity when configured | A single provider event contract is a v7 target. |
| Durable agent approval/input | Agent harness | The resumed run; approval itself is not retried | Waiting is a distinct paused error; denial and cancellation are not success | The pause and exact tool call are saved before returning; resume requires the run and approval/input identity | Keep and strengthen in v6. Hosts must use persistent storage for restart recovery. |
| Prompt-driven flow | Flow plus model provider | Provider/model behavior | `Execute` returns reply or error; context cancellation is an error | No ordered-step checkpoint boundary | Route adaptive tool use through the agent harness in v7; retain plain model transformations. |
| Ordered flow steps | Flow runner | Flow/step retry policy | Each step is `done` or `failed`; cancellation is currently represented as failure metadata | A shared run record saves completed steps; explicit `Resume` reuses them | Add a distinct cancelled/exhausted outcome compatibly where possible. |
| Agent-dispatch flow step | Flow owns step order; child agent owns adaptive work | Flow retries dispatch; agent owns its internal retries | RPC success completes the parent step; child run ID records lineage | Parent and child checkpoint independently | Prevent nested retries from multiplying effects; initially a v6 documentation/configuration fix. |
| Flow loop | Flow step owns the whole loop | Loop body and enclosing step policies | Reaching `LoopMax` currently returns the latest state as success | Only the enclosing step is checkpointed, so restart can repeat iterations | Persist progress and report exhaustion distinctly. Record-shape/outcome changes may require v7. |
| Broker-triggered flow | Broker callback invokes the flow | Broker-specific redelivery plus flow retry | The handler currently acknowledges by returning nil even when execution fails | Flow checkpoint can preserve ordered work, but acknowledgment is not tied to durable acceptance | Define acknowledgment after durable acceptance and deduplicate triggers; likely v7 for a portable broker contract. |

## Identity and outcome vocabulary

- **Run ID** identifies one agent or flow invocation. **Parent ID** and recorded
  child run IDs preserve dispatch lineage; **step name** identifies work inside
  an ordered run. Names must be unique within a flow.
- **Succeeded** means the invocation produced its declared result. It does not
  prove an external effect unless the service supplies evidence.
- **Failed** means execution ended with an error. Machine-readable error kinds
  should be preserved when available.
- **Cancelled** means the context ended by cancellation or deadline. It must not
  be reported as success.
- **Waiting** means durable input or approval is required. It is non-terminal and
  resumes only through an explicit matching operation.
- **Exhausted** means a retry, step, or loop budget ended before the requested
  condition was met. It is distinct from success and policy refusal.
- **Ambiguous effect** means an external action may have completed before its
  result was saved. Recovery must reconcile or reuse a stable idempotency key;
  blindly replaying is not safe.

Persisted status strings in v6 remain `running`, `waiting`, `done`, and `failed`
for compatibility. Cancellation and exhaustion can first be exposed through
error-kind metadata; a stored schema change requires versioning and migration.

## Decision record

1. **Flows own predefined order; agents own adaptive tool selection.** Plain
   service steps and model transformations remain available.
2. **The boundary that schedules a retry owns its budget.** A parent must not
   assume a child call is effect-free; retries need an idempotency key or an
   explicitly safe operation.
3. **Save before acknowledging durable acceptance.** Broker delivery must not be
   acknowledged merely because execution was attempted.
4. **Checkpoints prove recorded progress, not exactly-once effects.** Ambiguous
   external work is reconciled by the capability owner.
5. **Service-only applications remain first-class.** Shared outcome types must
   not introduce model or agent dependencies into RPC, registry, or broker APIs.

## Compatibility decisions

### Suitable for v6

- Preserve cancellation/error kinds and lineage consistently.
- Document retry ownership and require stable idempotency keys for effects.
- Add targeted restart tests around existing checkpoints and durable approval.
- Reject invalid resumes and make unsupported guarantees explicit.

### Reserved for v7 unless proved otherwise

- Make provider `Generate` one-turn-only and move tool iteration to the agent.
- Change portable broker acknowledgment semantics.
- Change persisted status/schema values or flow-loop record shape.
- Unify prompt-driven flow execution with the agent harness where public
  behavior changes.

The next gate is an executable fixture that restarts at each boundary, preserves
waiting state and lineage, injects duplicate/concurrent resume, models an
ambiguous effect, exhausts a budget, and proves an idempotent final operation is
not duplicated.
