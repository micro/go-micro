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

1. **Make AtlasCloud delegated notification side effects exact-once** ([#4163](https://github.com/micro/go-micro/issues/4163)) — the prior AtlasCloud schema/fallback fixes closed #4157, but the latest live provider-conformance signal shows a remaining Now-phase reliability seam: the `plan-delegate` harness can produce duplicate delegated `notify` side effects before the exact-once guard fails. Fixing this first keeps the same services → agents → workflows story portable across providers without changing public APIs.
2. **Add `micro loop` quickstart wayfinding to README and website docs** ([#4169](https://github.com/micro/go-micro/issues/4169)) — the blog now says the autonomous loop is shipped and reusable, but the lived developer on-ramp still depends on finding the launch post. Surface `micro loop init` / `micro loop verify`, token setup, branch protection, and CI-as-gate expectations in durable docs so the harness-building-itself story is adoptable, not just announced.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
