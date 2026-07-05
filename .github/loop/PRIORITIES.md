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

1. **Keep the first-agent examples map self-verifying** ([#4059](https://github.com/micro/go-micro/issues/4059)) — #4057 closed the stale top item by moving the website Getting Started page to the no-secret install → scaffold → run → call path, with README/CLI/website assertions now guarding that order. The next highest adoption seam is the examples layer: the README, website docs, and `examples/README.md` all tell a walkable first-agent story, but the examples map is not yet a first-class CI contract. Add a focused no-secret docs/harness assertion that the repository and website examples indexes lead through the same services → agents → workflows path (`hello-world`/0→1, `examples/first-agent`, `examples/support`/0→hero, debugging/inspection wayfinding) and fail on missing or stale links.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
