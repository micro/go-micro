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

1. **Fix AtlasCloud tool streaming capability mismatch** ([#4438](https://github.com/micro/go-micro/issues/4438)) — With the offline `micro new` 0→1 contract shipped, the top open risk is the provider capability/implementation seam: AtlasCloud still advertises streaming capability beyond its tool-streaming implementation, causing live agent streaming conformance to run an unsupported assertion. Fixing this keeps provider-backed chat and agent streaming honest without re-opening the no-secret adoption path.
2. **Broaden provider streaming conformance** ([#4386](https://github.com/micro/go-micro/issues/4386)) — Once the provider-specific AtlasCloud mismatch is resolved, expand the broader streaming matrix so chat, agent RPC, and A2A streaming regressions are caught across keyed providers while local CI continues to skip cleanly without secrets.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
