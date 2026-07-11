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

1. **Migrate store/postgres from pgx/v4 to pgx/v5** ([#4556](https://github.com/micro/go-micro/issues/4556)) — #4713 closed the deterministic no-secret plan/delegate resume contract, and the recent first-agent/doc wayfinding increments have covered the current adoption on-ramp. The highest-value remaining open work is now service-framework security upkeep: remove the pgx/v4 dependency/CVE drift without changing the public store API.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
