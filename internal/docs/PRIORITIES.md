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

1. **Add agent memory summarization and retrieval hooks** ([#3473](https://github.com/micro/go-micro/issues/3473)) — the Now-phase hardening queue is complete for the current pass: cross-provider dispatch, failure classification, checkpoint/resume trace coverage, RunInfo spans, and end-to-end A2A streaming all shipped and their issues are closed. The highest-value remaining roadmap gap is the Later-phase memory-management seam that makes long-running, looping agents practical: agents need deterministic summarization/retrieval over stored history so the services → agents → workflows lifecycle can continue across long contexts without turning into brittle prompt plumbing.

2. **Support A2A task resubscribe and input-required handoffs** ([#3474](https://github.com/micro/go-micro/issues/3474)) — after #3471 closed the main A2A streaming path, the next interop gap is live-operation continuity: remote agents must be able to reconnect to task streams and handle explicit `input-required` pauses. This ranks behind memory because it is a narrower protocol-depth item, but it is the next clear seam in making agents dependable neighbours over open protocols.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
