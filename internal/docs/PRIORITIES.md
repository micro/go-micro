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

## Developer experience (ranked)

1. **Maintain a real-world 0-to-hero reference example** ([#3284](https://github.com/micro/go-micro/issues/3284)) — with the hardening, agentic-depth, and deploy inner-loop contract items now shipped, the highest-value remaining gap is a living example that proves services, agents, flows, and interop compose into an actual system. This should double as documentation and CI smoke coverage so the lived story stays aligned with the README, website, and blog canon.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
