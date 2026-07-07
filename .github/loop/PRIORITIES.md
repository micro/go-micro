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

1. **Prevent duplicate delegated notifications in plan-delegate harness** ([#4245](https://github.com/micro/go-micro/issues/4245)) — Now-phase safety is the current red edge after the AtlasCloud 400 fallback shipped: the services → agents → workflows loop must not repeat delegated side effects while recovering or continuing a plan. Make the plan-delegate path idempotent and CI-verifiable before adding more depth.
2. **Make AtlasCloud guarded delegation pass reliably** ([#4244](https://github.com/micro/go-micro/issues/4244)) — keep cross-provider conformance close behind the side-effect fix: AtlasCloud now gets past the Minimax request-shape 400s, but the live agent harness still misses the required guarded delegate within the retry budget.
3. **Link examples wayfinding from website getting-started path** ([#4241](https://github.com/micro/go-micro/issues/4241)) — keep adoption weighted with hardening after the examples index shipped: the repo README and CLI now point at the first-agent/0→hero map, but go-micro.dev getting-started and quickstart pages should expose the same path with a CI-guarded docs smoke check.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
