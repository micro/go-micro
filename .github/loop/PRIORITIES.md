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

1. **Handle incomplete AtlasCloud agent-flow workspace tool calls** ([#4595](https://github.com/micro/go-micro/issues/4595)) — Highest-value open Now-phase hardening item after #4606 shipped in #4612: the event-driven agent-flow path observed the expected workspace/notification side effects once, suppressed duplicates, then still reported failure on an incomplete repaired workspace tool call. Fix this next so services → agents → workflows demos fail safely and consistently when a live provider emits malformed repaired tool calls.
2. **Make AtlasCloud universe A2A reachability probe deterministic** ([#4504](https://github.com/micro/go-micro/issues/4504)) — The remaining live AtlasCloud interop seam is the universe A2A reachability probe intermittently timing out after the checkout flow succeeds. Keep it behind the repaired-tool-call failure because it is side-effect safe and isolated to reachability/probe timing rather than the core 0→hero side-effect path.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
