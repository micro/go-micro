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

1. **Add a CI-verifiable agent verification loop** ([#3481](https://github.com/micro/go-micro/issues/3481)) — A2A task resubscribe/input-required handoffs have shipped, closing the last named Later A2A continuity seam for this pass. The highest-value open gap is now the verification/grader loop called out by the current canon: scheduled, looping, work-performing agents need a way to grade outputs, feed failures back through existing retry/supervision paths, and surface the result through RunInfo/trace boundaries without turning Go Micro into a graph DSL.

2. **Add a scheduled agent run harness contract** ([#3486](https://github.com/micro/go-micro/issues/3486)) — once verification exists, the next cohesive operational-harness step is to make the event/cadence side of “scheduled, looping, work-performing agents” CI-verifiable. This should compose existing services, agents, flows, store-backed memory, and run inspection so the scaffold → run → chat → inspect lifecycle extends to unattended work without adding a hosted scheduler or breaking public APIs.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
