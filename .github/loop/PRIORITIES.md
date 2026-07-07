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

1. **Add examples wayfinding index for first-agent adoption** ([#4223](https://github.com/micro/go-micro/issues/4223)) — keep developer adoption weighted with internal hardening: the README and getting-started guide now have a stronger first-agent path, but examples remain spread across docs, CLI output, and directories. A single CI-guarded examples map should make the smallest no-secret agent, the 0→hero support app, and next interop examples discoverable from one place.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
