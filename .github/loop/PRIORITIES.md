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

1. **Add CLI continuation for input-required agent runs** ([#4755](https://github.com/micro/go-micro/issues/4755)) — #4750 closed the cancellation/deadline propagation gap and there are no open codex PRs in flight. Do not re-queue more docs-link checks, AtlasCloud-specific text repair, or plan/delegate edge hardening for now; those areas have had several recent increments. The next highest-value user-facing gap is making human-in-the-loop pauses operable from the scaffold → run → chat → inspect path: list an `input-required` run, provide the missing input from the CLI, and inspect the completed run without requiring a developer to write a Go resume helper.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
