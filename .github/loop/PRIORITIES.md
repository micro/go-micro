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

1. **Lead CLI docs wayfinding with `micro agent demo`** ([#4046](https://github.com/micro/go-micro/issues/4046)) — #4044 shipped the README and website first-agent docs update for `micro agent demo`, so the queue should not keep #4041 open or re-ask for primary docs copy. The remaining adoption seam is inside the installed CLI: `micro docs` still starts from longer guide links rather than the new no-secret demo affordance, so the binary can drift from the public on-ramp a new developer just installed. Put `micro agent demo` first in the CLI docs wayfinding and cover it with the existing first-agent CLI boundary tests, preserving the scaffold → run → chat → inspect → deploy path without public API or positioning changes.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
