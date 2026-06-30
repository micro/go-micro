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

1. **Make long agent runs resumable from checkpoints** ([#3449](https://github.com/micro/go-micro/issues/3449)) — the flow grader loop shipped in #3443 and the run-trace analyzer shipped in #3447, closing the previous top queue item (#3439). With flow durability and optimization now proving the workflow side, the highest-value remaining lifecycle seam is agent-run durability: long agent loops should persist enough progress to resume without replaying completed tool side effects. This aligns the services → agents → workflows runtime by giving agents the same operable recovery posture that flows already have, while staying scoped to a non-breaking, CI-verifiable checkpoint/resume contract.

2. **Verification / grader loop for flows** ([#3435](https://github.com/micro/go-micro/issues/3435)) — founder-prioritized from the [loop-engineering](https://www.langchain.com/blog/the-art-of-loop-engineering) read: of the four loops (agent / verification / event-driven / hill-climbing), the harness provides the agent loop, event-driven flows, and the trace foundation for hill-climbing — but **verification** is the missing primitive. Add `flow.Verify(body, grader)` + `flow.LLMGrader(rubric)` that grade a step's output against a rubric and route failures back with feedback for a bounded retry, building on `flow.UntilLLM` and the existing retry/backoff. Closes the one gap between what we say (operable, trustworthy loops) and what the framework offers as a primitive.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
