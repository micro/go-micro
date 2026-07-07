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

1. **Link examples wayfinding from website getting-started path** ([#4241](https://github.com/micro/go-micro/issues/4241)) — top adoption gap after the examples index shipped and AtlasCloud notify follow-ups were fixed: the README and CLI now point at the first-agent/0→hero map, but go-micro.dev getting-started and quickstart still need the examples index/support reference links that make the no-secret on-ramp discoverable.
2. **Make AtlasCloud guarded delegation pass reliably** ([#4244](https://github.com/micro/go-micro/issues/4244)) — Now-phase cross-provider conformance remains important after the Minimax request-shape fallback, duplicate delegated-notification replay fixes, OpenAI-compatible text tool-call parsing, and AtlasCloud multi-step follow-up fixes shipped; keep it in queue until the live agent harness consistently observes the guarded delegate within the retry budget.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
