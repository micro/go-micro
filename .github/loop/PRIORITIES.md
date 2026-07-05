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

1. **Promote agent doctor in the first-agent debug loop** ([#4086](https://github.com/micro/go-micro/issues/4086)) — install troubleshooting and preflight now cover the before-run adoption seams; next close the after-run recovery seam for newcomers whose first agent starts but chat, gateway, registration, provider settings, or run history misbehave. Link `micro agent doctor` from the first-agent and debugging path, distinguish it from install troubleshooting and `micro agent preflight`, and add a focused CLI/docs harness assertion so the scaffold → run → chat → inspect recovery checkpoint stays discoverable.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
