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

1. **Document `micro agent demo` in the first-agent on-ramp** ([#4041](https://github.com/micro/go-micro/issues/4041)) — The last builder pass shipped `micro agent demo` in #4039, closing #4036/#4038 and giving the installed CLI a no-secret first-agent affordance. The queue should not re-ask for that command. The remaining adoption gap is that the canonical README/website first-agent path still leads users through examples and guides before naming the new CLI affordance, so the lived on-ramp can drift from what the binary now exposes. Put the command into the primary first-agent wayfinding and keep it covered by the existing first-agent docs tests, preserving the services → agents → workflows story without broad copy or API changes.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
