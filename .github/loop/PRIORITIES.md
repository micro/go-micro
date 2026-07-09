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

1. **Fix AtlasCloud plan-delegate incomplete delegate tool call** ([#4487](https://github.com/micro/go-micro/issues/4487)) — The adoption-facing inner-loop contract is now shipped, and there are no open `codex` PRs in flight, so the top remaining gap returns to a Now-phase resilience failure in the lived services → agents → workflows story: AtlasCloud can create task side effects, then stop on an incomplete repaired `delegate` tool call before notification/delegation completes. This is higher value than a single marker wording failure because it protects the cross-runtime handoff after real side effects and keeps the 0→hero/plan-delegate harness operable across providers.
2. **Fix AtlasCloud agent harness missing conformance marker** ([#4486](https://github.com/micro/go-micro/issues/4486)) — Once the delegate repair path is stable, close the remaining scheduled live-conformance gap where AtlasCloud's agent harness exhausts retries without emitting the required marker. This is still a Now-phase provider-conformance item, but narrower: it verifies the core agent/model+tool contract rather than the full services → agents → workflows handoff.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
