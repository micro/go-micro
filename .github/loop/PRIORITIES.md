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

1. **Add agent model retry/backoff contract** ([#4500](https://github.com/micro/go-micro/issues/4500)) — With the AtlasCloud marker gap and first-agent docs command parity now shipped, the highest-value remaining Now-phase hardening gap is provider-free coverage for transient model failures, retry budgets, backoff, and cancellation in the agent loop. This keeps cross-provider reliability moving without requiring live credentials or broad public-API changes.
2. **Add first-agent docs wayfinding link contract** ([#4508](https://github.com/micro/go-micro/issues/4508)) — The adoption story is now stronger, but the on-ramp spans README, website guides, CLI command output, examples, debugging, and 0→hero references. Add a provider-free check that fails on stale links or missing required wayfinding steps so the first-agent path remains walkable as docs and examples evolve.
3. **Stabilize AtlasCloud agent-flow onboarding side effects** ([#4503](https://github.com/micro/go-micro/issues/4503)) — The live AtlasCloud harness exposed a services → agents → workflows seam where a timeout can replay workspace creation without completing notification. Exact-once side-effect behavior is core to operable workflows, but it ranks after the provider-free retry contract and one adoption guardrail because it is live-provider-specific.
4. **Make AtlasCloud universe A2A reachability probe deterministic** ([#4504](https://github.com/micro/go-micro/issues/4504)) — The universe checkout flow completed, but the A2A reachability probe timed out under AtlasCloud. This matters for cross-framework operability and agent discoverability, yet it is narrower than the retry/idempotency issues because the durable workflow side effects already passed.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
