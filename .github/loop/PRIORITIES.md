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

1. **Add `micro loop` quickstart wayfinding to README and website docs** ([#4169](https://github.com/micro/go-micro/issues/4169)) — the loop launch post now says the autonomous harness is shipped and reusable, but the durable README/website on-ramp still does not make `micro loop init` / `micro loop verify`, token setup, branch protection, and CI-as-gate expectations easy to find. This is the highest-value adoption gap because the current goal weights a walkable on-ramp at least as highly as more internal hardening.
2. **Propagate cancellation and retry signals through provider model calls** ([#4175](https://github.com/micro/go-micro/issues/4175)) — after #4173 closed the duplicate delegated notification seam, the next Now-phase reliability gap is failure handling under real provider conditions: cancellation/deadline propagation and retry/backoff must not duplicate tool side effects. This keeps the same services → agents → workflows lifecycle dependable across providers without changing public APIs.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
