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

1. **Make first-agent docs wayfinding link targets self-verifying** ([#4064](https://github.com/micro/go-micro/issues/4064)) — #4062 closed the examples-map gap by making the repository and website examples indexes prove the services → agents → workflows path. The next highest developer-adoption risk is broken wayfinding rather than missing content: the README, CLI (`micro docs`, `micro agent demo`, scaffold next steps), website navigation, and examples map now point to the right first-agent/0→hero journey, but only some checks assert strings rather than resolving the target routes/files and command boundaries. Add a focused no-secret harness/docs assertion that fails when the first-agent path links to a missing local guide, stale website route, missing CLI command boundary, or out-of-order scaffold → run → chat → inspect → deploy step. Keep this as an adoption guardrail, not a public API or positioning change.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
