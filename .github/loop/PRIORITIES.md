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

1. **Verify installed first-agent on-ramp commands** ([#4197](https://github.com/micro/go-micro/issues/4197)) — developer adoption is the current goal, and the canonical story in the README, website, and recent blog posts depends on a provider-free path that works from the installed CLI. Keep `micro agent demo`, `micro examples`, `micro zero-to-hero`, first-agent docs, debugging, and 0→hero wayfinding aligned and CI-verifiable so 0→1 does not regress while hardening continues.
2. **Propagate cancellation and retry signals through provider model calls** ([#4175](https://github.com/micro/go-micro/issues/4175)) — with the AtlasCloud exact-once delegate/notify seam closed by #4200, the remaining Now-phase reliability gap is failure handling under real provider conditions: cancellation/deadline propagation and retry/backoff must not duplicate tool side effects. This keeps the services → agents → workflows lifecycle dependable across providers without changing public APIs.
3. **Resume durable agent runs from checkpoints** ([#4202](https://github.com/micro/go-micro/issues/4202)) — once the installed on-ramp and provider failure semantics are guarded, move into the Next-phase durable-agent-loop work: long-running agents should recover like flows, checkpoint progress, and avoid replaying completed tool side effects after interruption.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
