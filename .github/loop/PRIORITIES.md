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

1. **Stabilize plan-delegate harness against duplicate delegated notifications** ([#4118](https://github.com/micro/go-micro/issues/4118)) — the scheduled provider-conformance matrix is now in place, and its first live signal exposed an atlascloud/minimax duplicate-notify regression. Preserve the semantic “exactly one launch-readiness notification” contract with focused regression coverage so the live matrix remains a useful evaluator rather than a noisy gate.
2. **Add a copy/paste first-agent tutorial smoke harness** ([#4128](https://github.com/micro/go-micro/issues/4128)) — the CLI examples wayfinding shipped, so keep adoption pressure on the next most valuable seam: prove the website’s Your First Agent path can be followed from a clean workspace without relying on prose staying honest by hand. A focused CI-verifiable tutorial boundary keeps scaffold → run → chat → inspect cohesive after future CLI and docs changes.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
