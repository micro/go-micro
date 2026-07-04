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

1. **Add cross-provider agent conformance** ([#3901](https://github.com/micro/go-micro/issues/3901)) — #3895 closed the provider deadline/retry guidance gap and #3899 closed the AP2 foundation, leaving no open `codex` PRs or issues in flight. The highest-value Now-phase gap is therefore battle-testing the same first-agent behavior across providers: tool calling, multi-step runs, plan/delegate, and guardrails should fail loudly by provider instead of drifting behind a single happy-path backend. Keep default CI no-secret friendly with mock coverage and key-gated live providers.
2. **Checkpoint and resume durable agent runs** ([#3902](https://github.com/micro/go-micro/issues/3902)) — adoption now has install, scaffold, run, chat, inspect/history, and 0→hero coverage, but the lived harness story still has an asymmetry: flows checkpoint and resume while long agent runs do not. This is the top Next-phase agentic-depth gap because it makes services → agents → workflows feel like one runtime under interruption and deploy/restart conditions. Keep it additive; do not break the public agent API.
3. **Broaden provider streaming and keep chat/A2A streaming end to end** ([#3903](https://github.com/micro/go-micro/issues/3903)) — after conformance and durability, streaming is the next developer-visible seam in the inner loop. Real chat and long-running A2A tasks need token streaming to stay coherent from provider → `ai.Stream` → `micro chat` → A2A `message/stream`, with mock/default CI coverage plus key-gated live provider checks and safe fallback for non-streaming providers.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
