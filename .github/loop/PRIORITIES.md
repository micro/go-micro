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

1. **Trace agent runs as OpenTelemetry spans** ([#3908](https://github.com/micro/go-micro/issues/3908)) — Recent builder work closed the AtlasCloud `plan-delegate` continuation gap (#4013 via #4019) and the earlier guarded-delegate seam (#4007 via #4015). The only open `codex` PR is #4022, a human-review blog/changelog draft that is intentionally not auto-merged and does not change the autonomous builder queue. With the Now-phase live-provider conformance seam shipped, the highest-value Next-phase gap is developer-visible operability: `RunInfo` steps, tool calls, delegation, status, duration, and failures should correlate to OpenTelemetry spans while defaulting to no-op when tracing is not configured. This directly supports the scaffold → run → chat → inspect → deploy inner loop and keeps services → agents → workflows observable as one runtime.
_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
