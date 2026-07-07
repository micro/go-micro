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

1. **CI-verify the first-agent 0-to-hero contract** ([#4272](https://github.com/micro/go-micro/issues/4272)) — Current goal is developer adoption, and the README/website now describe a strong first-agent path after the examples wayfinding work (#4239/#4266), but the highest-value next increment is making that on-ramp executable as a provider-free contract. Prove scaffold → run → chat → inspect → deploy dry-run against the maintained examples so services → agents → workflows is something a new developer can walk, not just read.
2. **Propagate cancellation and retry signals through provider model calls** ([#4273](https://github.com/micro/go-micro/issues/4273)) — With the AtlasCloud guarded-delegation failures closed by #4270, the next Now-phase reliability seam is real-provider failure handling: context deadlines, cancellation, retry/backoff, and rate limits must not duplicate completed tool side effects. Keep this behind focused tests/fakes and avoid public API changes.
3. **Add an agent debugging quickcheck to the CLI harness** ([#4274](https://github.com/micro/go-micro/issues/4274)) — The on-ramp is only credible if the first surprising agent run is inspectable. Add provider-free harness coverage for the documented `micro inspect agent`/run-inspection path and keep README/website debugging wayfinding aligned, so the inner loop does not stop at “it ran” but reaches “I can understand what happened.”

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
