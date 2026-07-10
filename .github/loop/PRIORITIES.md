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

1. **Add a no-secret first-agent chat transcript check** ([#4618](https://github.com/micro/go-micro/issues/4618)) — Developer adoption is now the highest-value open Now-phase item after #4504 shipped in #4621. README, the website, and the v6.3.15 blog point at the provider-free first-agent path, but the 0→1 agent experience still needs an expected chat/tool-call transcript that a new developer can compare against before adding provider keys. Make the smallest first-agent path more walkable and CI-verifiable.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
