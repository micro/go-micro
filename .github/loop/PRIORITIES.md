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

1. **Make micro new zero-to-one contract offline** ([#4463](https://github.com/micro/go-micro/issues/4463)) — The current developer-adoption gate can fail before a new user reaches an agent: `micro new` runs `go mod tidy` against the public Go proxy before the contract test rewrites the module to the local checkout. Fix this first because the no-secret scaffold → run → call path is the 0→1 contract and must be deterministic in CI and constrained environments.
2. **Fix AtlasCloud tool streaming capability mismatch** ([#4438](https://github.com/micro/go-micro/issues/4438)) — The provider matrix still advertises AtlasCloud streaming capability beyond its tool-streaming implementation, causing the live agent streaming conformance path to run an unsupported assertion. Fixing the capability/implementation seam keeps streaming honest without blocking no-secret adoption work.
3. **Broaden provider streaming conformance** ([#4386](https://github.com/micro/go-micro/issues/4386)) — Once the provider-specific AtlasCloud failures are resolved, expand the broader streaming matrix so chat, agent RPC, and A2A streaming regressions are caught across keyed providers while local CI continues to skip cleanly without secrets.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
