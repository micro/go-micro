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

1. **Gate mock provider plan-delegate resume scenarios** ([#4713](https://github.com/micro/go-micro/issues/4713)) — #4709 closed the live AtlasCloud nested tool-call rejection gap and #4725 closed the website first-agent wayfinding gap, so the highest-value remaining Now work is deterministic no-secret agent-loop coverage. Completed plan steps, notifications, unsafe fallback parsing, and resume semantics must stay stable without provider keys before broadening into live-provider conformance.
2. **Migrate store/postgres from pgx/v4 to pgx/v5** ([#4556](https://github.com/micro/go-micro/issues/4556)) — Security upkeep matters to the service-framework half of the harness, and #4556 is the remaining open enhancement/security item. Keep it below the agent-loop contract because it is narrower than the current developer-adoption goal, but do not let reachable dependency risk drift indefinitely.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
