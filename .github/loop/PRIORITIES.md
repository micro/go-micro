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

1. **Harden AtlasCloud guarded-delegate conformance** ([#4183](https://github.com/micro/go-micro/issues/4183)) — #4185 closed the duplicate delegated notification gap from #4163, and the latest live provider run now fails one step earlier in the same Now-phase adoption seam: AtlasCloud completes the required echo tool call but does not reliably exercise the refused guarded `delegate` path. Make the conformance prompt/retry/fallback path require both tool calls before accepting a final answer so the scheduled evaluator reflects real services → agents → workflows behavior instead of a provider-specific shortcut.
2. **Propagate cancellation and retry signals through provider model calls** ([#4175](https://github.com/micro/go-micro/issues/4175)) — once the live conformance harness again proves the guarded delegate path, the next Now-phase reliability gap is failure handling under real provider conditions: cancellation/deadline propagation and retry/backoff must not duplicate tool side effects. This keeps the same services → agents → workflows lifecycle dependable across providers without changing public APIs.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
