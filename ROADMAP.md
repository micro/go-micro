# Go Micro Roadmap

Go Micro is an **agent harness** and service framework for Go. A harness is the
runtime around an agent — the tools, memory, guardrails, workflows, state,
discovery, and protocols it needs to operate a system rather than just answer a
prompt. An agent is a distributed system — it discovers services, calls them,
holds state, and recovers from failure — so the harness is the runtime services
already have, and building an agent is building a service. The roadmap has two
jobs: make **agentic development** excellent, and make the **developer experience**
around it excellent.

The full, current roadmap lives at **[go-micro.dev/docs/roadmap](https://go-micro.dev/docs/roadmap)**
([source](internal/website/docs/roadmap.md)). The highlights:

## Where we are (v6)

Services, agents (`plan`/`delegate`, guardrails, memory, tool middleware), durable
flows, the MCP and A2A gateways (both directions), x402 paid tools, secure by
default.

## Principles

1. Build into what people run, never a separate product (no hosted platform, no
   enterprise edition, no VC).
2. CLI-first — the CLI is the experience; UI must earn its place, never bloat.
3. The getting-started flow is a contract: *0→1* (scaffold → run → call) and
   *0→hero* (a working multi-agent system) must always work and are verified on
   every change.
4. Interaction matters as much as running — chatting with agents, inspecting runs
   and history, end to end.
5. Battle-tested: works across every provider, fails safely, observable.

## Now — hardening

- **Cross-provider conformance** — the same agent scenario across all seven
  providers, gated on keys, on a schedule.
- **Failure & resilience** — timeouts, rate limits, cancellation, deadline/context
  propagation, retry/backoff.
- **Getting-started contract** — define and CI-verify the 0→1 and 0→hero flows.

## Next — agentic depth

- **Durable agent loop** — resume a long run via `Checkpoint` (flows already do).
- **Streaming** — `ai.Stream` + A2A `message/stream`, end to end.
- **Agent observability** — `RunInfo` → OpenTelemetry spans.

## Later

- Memory management (summarization, retrieval/RAG); human-in-the-loop pause/resume;
  x402 live-facilitator conformance and paid remote tools with spend caps; A2A
  streaming, push notifications, and multi-turn tasks.

## Developer experience (ongoing)

- A seamless CLI inner loop (scaffold → run → chat → inspect → deploy); UI
  discipline (trim what isn't great); a maintained real-world example that doubles
  as the 0→hero reference; docs kept in lockstep with the code.

## How it's sustained

The framework is the product, funded by sponsorship from those who run it — not a
hosted service, enterprise tier, or venture funding. See
[the v6 story](https://go-micro.dev/blog/27).

## Contributing & feedback

Pick an item, open an issue to discuss the approach, and submit a PR. Or join the
[Discord](https://discord.gg/WeMU5AGxD). Include tests, run `make test` and
`make lint`.

## Version support

- **v6** — active development (current).
- **v5** — security fixes only.
- **v4 and earlier** — end of life.

Major versions (v5 → v6) carry breaking changes; minors are backward-compatible.
See the [v5 → v6 migration guide](https://go-micro.dev/docs/guides/migration/v5-to-v6).
