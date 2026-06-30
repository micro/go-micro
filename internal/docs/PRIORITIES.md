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

1. **Broaden end-to-end agent streaming coverage** ([#3402](https://github.com/micro/go-micro/issues/3402)) — with durable agent checkpoint/resume guarded by #3406, streaming is now the highest-value Next-phase developer-visible seam: provider tokens need to move consistently through `ai.Stream`, `micro chat`, `Agent.Chat`, and A2A streaming. CI-safe local coverage plus provider-gated conformance should lock down chunk ordering, terminal/error events, and fallback behavior.
2. **Emit OpenTelemetry spans for agent runs** ([#3403](https://github.com/micro/go-micro/issues/3403)) — once long runs can resume and stream, operators need the same run story in traces that developers see via `micro runs`: lifecycle boundaries, model/tool calls, approvals, retries, failures, and cancellation correlated by run ID without leaking sensitive payloads.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
