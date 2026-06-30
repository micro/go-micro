# Priorities

The ranked work queue for the autonomous improvement loop. The
**architecture-review** pass (the *architect*) owns this file: each run it turns
the [roadmap](../../ROADMAP.md) plus an internal scan (gaps in the
services → agents → workflows lifecycle, API coherence, drift, tech debt, test and
DX friction) into a single ordered list — highest-value first — and links each
item to a tracking issue. The hourly **continuous-improvement** pass works the
**top item whose issue is still open**. So the architect decides *what*, and the
increment loop *builds* it.

**Reading / editing.** An item is done when its linked issue closes (the increment
that builds it adds `Closes #<issue>`). Roadmap phase (Now → Next → Later) is the
primary ordering; internal findings are interleaved by value, not kept in a
separate list. The human can reorder this list — or the issues — at any time to
redirect the loop; direction always wins.

**Off-limits to the loop** (the architect proposes these as notes, never as queue
items the loop can auto-merge): brand/positioning copy, breaking public-API
changes, architectural rewrites. Those go to the human.

## Work queue (ranked)

1. **Support A2A task resubscribe and input-required handoffs** ([#3474](https://github.com/micro/go-micro/issues/3474)) — the earlier Now/Next hardening seams are closed for this pass: cross-provider dispatch, failure classification, checkpoint/resume trace coverage, RunInfo spans, end-to-end A2A streaming, and memory summarization/retrieval hooks have shipped. The highest-value remaining roadmap gap is live-operation continuity over A2A: remote agents must be able to reconnect to task streams and handle explicit `input-required` pauses so Go Micro agents remain dependable neighbours over open protocols.

2. **Add a CI-verifiable agent verification loop** ([#3481](https://github.com/micro/go-micro/issues/3481)) — blog/32 makes the next strategic frontier explicit: scheduled, looping, work-performing agents need a verification/grader loop around the agent loop. This ranks just behind the open A2A continuity issue because it is an internal operational-harness primitive rather than a named roadmap item, but it is the next cohesive step after RunInfo spans, failure semantics, and memory: grade outputs, route feedback through existing retry/supervision paths, and make dependable agent work observable without drifting into a graph DSL.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
