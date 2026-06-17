# Durable Execution: Flow Steps & Checkpoint

**Status:** Design proposal — not yet implemented.

This note sketches two related changes:

1. Give **flow** a real step model — a flow is a *task* made of *ordered
   steps* — so it becomes the deterministic-workflow engine it has always
   claimed to be (today it runs a single LLM step per event).
2. Introduce **`Checkpoint`**, a pluggable durability primitive that
   persists run progress and resumes after a crash. Store-backed by
   default; both flow and agent use it.

The two are designed together because a step boundary is the natural
place to checkpoint.

---

## Motivation

A flow or agent run is long, expensive, and has side effects partway
through (it sent an email at step 2, charged via x402 at step 4). Today
all in-flight state lives in process memory: a crash loses the run, and
re-running from the top repeats the side effects.

Durable execution means the run survives a crash and **continues from
where it stopped**, without re-doing completed steps.

This is squarely a distributed-systems concern — checkpoint state, replay
on restart, pluggable backend — i.e. go-micro's kind of problem, built on
primitives it already has (`store`, `WrapTool`, `call.ID`).

---

## What flow is today (for contrast)

`flow` is a concrete `*Flow` struct. Per broker event, `Execute` runs
**one** augmented-LLM turn (a single `Generate` with services as tools)
or dispatches the event to an agent, records one `Result`, and returns.
There is no notion of a task with ordered steps, no carried state, no
checkpoint. The step model below generalizes today's behavior: a flow
with one LLM step == current flow.

---

## Core concepts

### Step

The unit of a flow. A step takes the carried state, does one thing, and
returns the new state. Idiomatic to go-micro: a handler func, with
built-in constructors for the common kinds.

```go
// A Step is one unit of work in a flow.
type Step struct {
    Name string
    Run  func(ctx context.Context, s State) (State, error)
}
```

Built-in step constructors cover the common actions (each is just a
`Step` with a prepared `Run`):

```go
flow.LLMStep(name, opts...)            // one augmented-LLM turn
flow.CallStep(name, service, endpoint) // one RPC to a service
flow.AgentStep(name, agentName)        // dispatch to a registered agent
flow.FuncStep(name, fn)                // arbitrary developer function
```

Steps are **authored by the developer** and run in order. That is the
defining difference from an agent, where the *model* chooses the steps.

### State

What carries across steps — JSON-serializable so it can be checkpointed.

```go
type State struct {
    Data map[string]any
}

func (s State) Get(key string) any
func (s State) Set(key string, v any) State
func (s State) Scan(v any) error   // decode Data into a typed struct
```

The trigger event seeds `State` (e.g. `Data["trigger"] = <event body>`).
Each step reads what it needs and returns an updated `State`.

### Run

The persisted record of one in-flight execution — what `Checkpoint`
saves and loads.

```go
type Run struct {
    ID      string        // durable run id (idempotency root)
    Flow    string        // flow name
    Step    int           // index of the NEXT step to run
    State   State         // carried data
    Steps   []StepRecord  // per-step status + outcome
    Started time.Time
    Updated time.Time
    Done    bool
}

type StepRecord struct {
    Name   string
    Status string // pending | in_progress | done | failed
    Result string // short serialized outcome / summary
    Error  string
}
```

### Checkpoint

The pluggable durability primitive. Persists and restores a `Run`.

```go
type Checkpoint interface {
    Save(ctx context.Context, run Run) error
    Load(ctx context.Context, runID string) (Run, bool, error)
    Delete(ctx context.Context, runID string) error
}
```

The built-in implementation is **store-backed** and on by default,
keyed in the store:

```
flow/{name}/run/{runID}   →   JSON(Run)
```

Because it rides on `store.Store`, the *storage* is already pluggable
(Postgres, NATS KV, file) with no extra interface.

---

## The run loop

```
run := load(runID) or new Run{Step: 0, State: seed}

for i := run.Step; i < len(steps); i++ {
    run.Steps[i].Status = "in_progress"; checkpoint.Save(run)

    out, err := steps[i].Run(ctx, run.State)
    if err != nil {
        run.Steps[i].Status = "failed"; run.Steps[i].Error = err
        checkpoint.Save(run)
        return err            // resumable: a later retry resumes at i
    }

    run.State = out
    run.Steps[i].Status = "done"
    run.Step = i + 1          // <-- checkpoint boundary
    checkpoint.Save(run)
}

run.Done = true; checkpoint.Save(run)   // (or Delete on success)
```

On restart, `Load` returns the `Run`; the loop starts at `run.Step`, so
completed steps are skipped — their effects already happened and their
output is already in `run.State`.

### Idempotency (the honest part)

True exactly-once is impossible if a crash lands *inside* a step. What we
provide is at-least-once + a stable **idempotency key** per step:
`runID + stepName`. That key is passed to the tool as `call.ID`, so a
replayed call is recognized downstream and de-duplicated. Side-effecting
steps must cooperate (honor the key). The framework makes this consistent;
it cannot make it free.

This is where the existing `WrapTool` seam pays off: a durable wrapper
checks the checkpoint — if this `call.ID` already has a recorded result,
return it without re-calling.

---

## Agent reuse

The agent loop is the **self-directed** analogue and uses the same
`Checkpoint`. The difference is who authors the steps:

| | Steps authored by | Steps known | Durability |
|---|---|---|---|
| **flow** | developer | up front (ordered list) | checkpoint between steps |
| **agent** | the model | discovered at runtime | checkpoint each LLM turn + its tool calls |

For the agent, `Run.Steps` grows as the model acts, instead of being
predefined. One requirement: the agent must own its loop (today the
provider drives it), so it can `Save` between turns. That is the one
structural change on the agent side.

---

## Pluggability — two levels

1. **Storage (free today).** Built-in `Checkpoint` over `store.Store`;
   swap the store backend. Covers "checkpoint to my DB instead."
2. **Engine (future).** Because steps are now explicit and named, a flow
   can be mapped onto an external durable-execution engine — each `Step`
   becomes a Temporal activity / Restate handler — by providing an
   alternative runner. Most users only need level 1; level 2 exists so
   teams already running Temporal aren't forced off it.

The explicit step model is what makes level 2 possible later; we don't
build it now.

---

## Proposed API

```go
f := flow.New("onboard-user",
    flow.Trigger("events.user.created"),
    flow.Steps(
        flow.LLMStep("plan", flow.Prompt("Plan onboarding for {{.trigger}}")),
        flow.CallStep("workspace", "workspace", "Workspace.Create"),
        flow.AgentStep("welcome", "comms"),
    ),
    // Durable by default (store-backed). Swap to plug in another backend.
    flow.WithCheckpoint(flow.StoreCheckpoint(service.Options().Store)),
)
f.Register(reg, broker, client)
```

A single-step flow keeps today's behavior, so this is additive.

---

## Scope & phasing

1. **Step model in flow** (no durability yet): `Step`, `State`, ordered
   `Steps`, the run loop. Single-step flows unchanged.
2. **`Checkpoint` + store-backed default**: persist/resume flow runs.
3. **Agent durability**: move the agent loop in-package, reuse
   `Checkpoint`. Opt-in (`AgentDurable()`), default off — overkill for
   short interactive chats, essential for long unattended runs.
4. **Engine-level pluggability** (Temporal/Restate): only if demand.

Each phase is independently useful and shippable.

---

## Open questions

- **State shape:** `map[string]any` (simple, JSON) vs a typed/byte
  payload per step. Map is easier; typed is safer.
- **Success cleanup:** `Delete` the run on completion vs keep it for
  audit/history (flow already records `Result`s).
- **Retry policy:** is per-step retry a `Step` concern (a wrapper) or a
  flow-level option?
- **Naming of step constructors:** `LLMStep`/`CallStep`/`AgentStep`
  vs a single `Step{Kind, ...}`.
