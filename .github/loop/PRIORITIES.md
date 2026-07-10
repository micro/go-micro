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

1. **Add a 0→hero inspect transcript check** ([#4627](https://github.com/micro/go-micro/issues/4627)) — The no-secret first-agent transcript shipped in #4625 and closed #4618, so the next highest-value Now-phase adoption gap is the larger 0→hero contract. README, docs, and the v6.3.15 blog now make the provider-free first-agent path discoverable; the next seam is proving scaffold → run → chat → inspect → workflows with deterministic output a developer can compare against before provider keys. Keep this focused on the maintained support example/harness and CI-verify the transcript so the services → agents → workflows lifecycle stays walkable.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
