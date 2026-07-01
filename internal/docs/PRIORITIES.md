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

1. **Polish the 0-to-hero reference app** ([#3515](https://github.com/micro/go-micro/issues/3515)) — the latest increment shipped retrieval-backed agent memory, closing the last listed memory-management gap. With the core services → agents → workflows primitives now present (scheduled/durable runs, RunInfo tracing, provider stream conformance, A2A resubscribe/input-required, durable checkpoints, and retrieval memory), the highest-value remaining gap is coherence: one maintained, no-secret, CI-verifiable 0→hero reference that proves the whole lifecycle and CLI inner loop from scaffold → run → chat → inspect → deploy. This is the roadmap's ongoing DX contract and the best way to prevent the harness from feeling like separate demos rather than one runtime.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
