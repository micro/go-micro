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

1. **Flow hill-climbing loop from run traces** ([#3439](https://github.com/micro/go-micro/issues/3439)) — verification/grader flows shipped in #3443, and there are no open codex PRs currently carrying follow-up work. The highest-value remaining lifecycle gap is to use the run/trace foundation plus the new grader signal to analyze failures over time and propose prompt/grader improvements. This keeps the services → agents → workflows story operable by turning durable, observable runs into a product-facing proof of continuous workflow improvement rather than leaving evaluation as a one-off step.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
