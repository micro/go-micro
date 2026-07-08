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

1. **CI-verify the Your First Agent chat walkthrough** ([#4363](https://github.com/micro/go-micro/issues/4363)) — Developer adoption is the current goal, and the README/website now promise a walkable first-agent path. With the AtlasCloud Minimax follow-up retry fixed by #4366 and #4355 closed, the highest-value Now-phase gap is keeping the documented `micro chat` plus inspect/history on-ramp executable in no-secret CI so 0→1 agent success does not drift.
2. **Trace agent RunInfo in OpenTelemetry spans** ([#4315](https://github.com/micro/go-micro/issues/4315)) — Once the first-agent walkthrough contract is stable, the highest Next-phase operability gap is connecting existing run metadata to traces so real agent runs can be debugged across steps, tool calls, delegation, failures, services, and flows without inventing a new surface.
3. **Resume agent runs from checkpoints** ([#4368](https://github.com/micro/go-micro/issues/4368)) — Durable agent runs are the next lifecycle seam after observability: flows already checkpoint, and agents need a focused, non-breaking resume slice that preserves completed tool calls and avoids duplicate side effects before broader durability or API design work.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
