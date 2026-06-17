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
with one step == current flow.

---

## Core concepts

### State

What carries across steps. **A struct, not a map** — a typed `Payload`
plus a `Stage` marker so you can always tell where a run is.

```go
type State struct {
    Stage   string // name of the step the run is at — where it is
    Payload []byte // carried data, serialized; use Set / Scan
}

// Set replaces the payload with the JSON encoding of v.
func (s *State) Set(v any) error
// Scan decodes the payload into v (a pointer to the caller's struct).
func (s State) Scan(v any) error
```

The developer defines their own payload struct and threads it through
with `Set`/`Scan` — type-safe at the edges, serializable in the middle
(which is what makes checkpointing possible). `Stage` is the readable
"where am I"; the engine also uses it as the resume point.

The trigger event seeds the first `State`.

### Step

The unit of a flow. **One kind** — a struct with a name, the action to
run, and an optional retry override. No per-kind constructors.

```go
type StepFunc func(ctx context.Context, in State) (State, error)

type Step struct {
    Name  string
    Run   StepFunc
    Retry int // optional per-step override of the flow's retry (0 = use flow default)
}
```

Common actions are **helpers that return a `StepFunc`**, dropped into
`Step.Run` — so there is still one `Step` type, and the actions compose:

```go
flow.Call(service, endpoint) StepFunc // one RPC to a service
flow.LLM(opts...)           StepFunc // one augmented-LLM turn
flow.Agent(name)            StepFunc // dispatch to a registered agent
// …or write your own StepFunc.
```

Steps are **authored by the developer** and run in order. That ordering
is the defining difference from an agent, where the *model* chooses the
steps.

### Run

The persisted record of one execution — what `Checkpoint` saves and
loads. Retained for success and failure (see retention below).

```go
type Run struct {
    ID      string       // durable run id (idempotency root)
    Flow    string       // flow name
    State   State        // carried data + Stage (where it is)
    Steps   []StepRecord // per-step status + outcome (history/audit)
    Status  string       // running | done | failed
    Started time.Time
    Updated time.Time
}

type StepRecord struct {
    Name    string
    Status  string // pending | in_progress | done | failed
    Attempts int   // how many tries this step took
    Result  string // short serialized outcome / summary
    Error   string
}
```

The resume point is `State.Stage` — there is no separate numeric cursor,
so there is one source of truth for "where it is."

### Checkpoint

The pluggable durability primitive. Persists and restores a `Run`.

```go
type Checkpoint interface {
    Save(ctx context.Context, run Run) error
    Load(ctx context.Context, runID string) (Run, bool, error)
    Delete(ctx context.Context, runID string) error
}
```

The built-in implementation is **store-backed** and on by default, keyed
in the store:

```
flow/{name}/run/{runID}   →   JSON(Run)
```

Because it rides on `store.Store`, the *storage* is already pluggable
(Postgres, NATS KV, file) with no extra interface.

**Retention:** completed runs (success *and* failure) are **kept** by
default, so you have a durable history of what ran. `Delete` is only
called when the flow opts in with `flow.DeleteOnSuccess()` (failures are
always kept).

---

## The run loop

```
run := load(runID) or new Run{State: {Stage: steps[0].Name, ...}}

start := index of step named run.State.Stage
for i := start; i < len(steps); i++ {
    step := steps[i]
    run.Steps[i].Status = "in_progress"; checkpoint.Save(run)

    out, err := runWithRetry(ctx, step, run.State, retriesFor(step))
    run.Steps[i].Attempts = attemptsTaken
    if err != nil {
        run.Steps[i].Status = "failed"; run.Steps[i].Error = err
        run.Status = "failed"; checkpoint.Save(run)   // kept for audit
        return err                                    // resumable: retry resumes here
    }

    run.State = out
    run.Steps[i].Status = "done"
    if i+1 < len(steps) {
        run.State.Stage = steps[i+1].Name             // <-- checkpoint boundary
    } else {
        run.State.Stage = ""                          // finished
    }
    checkpoint.Save(run)
}

run.Status = "done"; checkpoint.Save(run)
// Delete only if flow.DeleteOnSuccess() was set.
```

On restart, `Load` returns the `Run`; the loop resumes at the step named
`run.State.Stage`, so completed steps are skipped — their effects already
happened and their output is already in `run.State.Payload`.

### Retry

Flow-level by default, per-step override when needed (e.g. a tool that
times out):

```go
flow.Retry(2)                 // flow-level default for every step
flow.Step{Name: "charge", Run: …, Retry: 0}  // override: never retry this one
```

`retriesFor(step)` uses `step.Retry` if set, else the flow default.

### Idempotency (the honest part)

True exactly-once is impossible if a crash lands *inside* a step. What we
provide is at-least-once + a stable **idempotency key** per step:
`runID + stepName`. That key is passed to the tool as `call.ID`, so a
replayed call is recognized downstream and de-duplicated. Side-effecting
steps must cooperate (honor the key). The framework makes this
consistent; it cannot make it free.

Retry uses the same key, so a retried step is de-duplicated the same way.
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
type Onboarding struct {
    Email       string `json:"email"`
    WorkspaceID string `json:"workspace_id"`
}

f := flow.New("onboard-user",
    flow.Trigger("events.user.created"),
    flow.Retry(2), // flow-level retry default
    flow.Steps(
        flow.Step{Name: "plan", Run: flow.LLM(flow.Prompt("Plan onboarding for {{.Email}}"))},
        flow.Step{Name: "workspace", Run: flow.Call("workspace", "Workspace.Create")},
        flow.Step{Name: "welcome", Run: flow.Agent("comms")},
    ),
    // Durable by default (store-backed); runs are retained for audit.
    flow.WithCheckpoint(flow.StoreCheckpoint(service.Options().Store)),
)
f.Register(reg, broker, client)
```

A single-step flow keeps today's behavior, so this is additive.

---

## Decisions (resolved)

- **State is a struct, not a map** — typed `Payload` + `Stage`. The
  developer defines the payload struct; `Stage` doubles as the resume
  point, so there is one source of truth for position.
- **One `Step` kind** — a struct with `Name`, `Run`, and an optional
  `Retry`. Common actions are `StepFunc` helpers (`Call`, `LLM`,
  `Agent`), not separate step constructors.
- **Runs are retained** for success and failure by default;
  `flow.DeleteOnSuccess()` opts into cleanup (failures always kept).
- **Retry is a flow-level option** (`flow.Retry(n)`), with a per-step
  `Retry` field as a fine-grained override.

---

## Scope & phasing

1. **Step model in flow** (no durability yet): `State`, `Step`, ordered
   `Steps`, the run loop, retry. Single-step flows unchanged.
2. **`Checkpoint` + store-backed default**: persist/resume flow runs,
   retention.
3. **Agent durability**: move the agent loop in-package, reuse
   `Checkpoint`. Opt-in (`AgentDurable()`), default off — overkill for
   short interactive chats, essential for long unattended runs.
4. **Engine-level pluggability** (Temporal/Restate): only if demand.

Each phase is independently useful and shippable.
