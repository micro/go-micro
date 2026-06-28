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

1. **Agent memory compaction and retrieval controls** (#3209) — add explicit,
   store-backed summarization/retrieval behavior for long-running agents so
   durable, streaming, observable runs do not become context-growth demos.
   Roadmap → memory management; ranked first because the Now/Next reliability
   work has shipped and long-lived agents need bounded, recoverable context before
   deeper autonomous workflows can be trustworthy.
2. **Human-in-the-loop pause/resume for agent runs** (#3210) — persist a paused
   approval/intervention point and resume safely through the existing
   guardrail/run-inspection path. Roadmap → human-in-the-loop pause/resume; ranked
   second because it turns guardrails from one-shot refusals into an operable
   workflow primitive without changing the public architecture.
3. **x402 spend caps and live facilitator conformance** (#3211) — enforce paid-tool
   budgets and add credential-gated live facilitator checks. Roadmap → x402 paid
   remote tools; ranked after memory and pause/resume because it hardens a narrower
   paid-tool seam but is important for trust once agents can act for longer.
4. **A2A push notifications and multi-turn task support** (#3212) — extend the A2A
   gateway/client path beyond streaming lifecycle updates into push and multi-turn
   task state. Roadmap → A2A push notifications and multi-turn tasks; ranked after
   local run operability because interop depth compounds best once the local run
   lifecycle is bounded and inspectable.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
