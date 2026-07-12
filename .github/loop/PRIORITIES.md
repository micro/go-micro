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

1. **Stream remote agent replies through `micro chat`** ([#4760](https://github.com/micro/go-micro/issues/4760)) — #4758 closed the input-required CLI continuation gap, so do not re-queue more resume breadcrumbs, docs-link guards, AtlasCloud-specific text repair, or plan/delegate edge hardening for now. The next highest-value user-facing gap is making the scaffold → run → chat → inspect loop feel live when `micro chat --stream` talks to a registered agent, not only when it uses the direct service/tool fallback: stream chunks from a stream-capable agent, fall back cleanly to `Agent.Chat`, and keep inspectable run history coherent.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
