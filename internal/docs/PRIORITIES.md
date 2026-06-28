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

## Later (ranked)

1. **Add human-in-the-loop pause and resume for agent runs** ([#3278](https://github.com/micro/go-micro/issues/3278)) — after memory summarization/retrieval shipped, the highest-value remaining roadmap gap is making long-running agents safer and more operable by letting a run enter an input-required pause, persist enough state to survive the boundary, and resume with human input using the existing agent/store/flow/A2A semantics rather than a separate orchestration layer.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
