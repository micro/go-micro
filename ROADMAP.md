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

Services, agents (`plan`/`delegate`, guardrails, memory, tool middleware,
checkpoint/resume, and OpenTelemetry run spans), durable flows, the MCP and A2A
gateways (both directions, including A2A streaming,
push notifications, and multi-turn continuation), x402 paid tools, secure by
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

The forward work is **net-new capability**, not more hardening. Maintenance
(conformance, resilience, DX polish) continues in the background (see *Ongoing*
below) — but it is not the roadmap. This capability work is.

## Now — capability

- **Agents that pay (x402 buyer in the runtime).** The seller side ships (paid
  tools via the `wrapper/x402` middleware) and the buyer `x402.Client` (a
  budget-capped `Payer` that turns a `402` into pay-and-retry) exists — but an
  agent can't yet *autonomously* pay for a paid tool. Wire the buyer into the
  agent tool loop: a budget-capped `AgentPayer` so an agent that hits a
  payment-required tool settles it within budget and retries, with the spend
  gated (like `ApproveTool`) and observable in `RunInfo`/traces. This makes
  go-micro a runtime for **autonomous agent commerce**. *(flagship — decomposed
  into issues in the loop queue)*
- **AP2 mandate foundation** ([#3552](https://github.com/micro/go-micro/issues/3552))
  — verifiable payment **mandates** (a Checkout Mandate and a Payment Mandate),
  signed and attached over A2A, with the Payment Mandate naming an x402 rail. The
  authorization/audit layer above A2A + x402 that positions go-micro early in the
  emerging agent-payments standard (Google's AP2, standardized via FIDO).
  Additive and opt-in.

## Next — reach & deployment

- **gRPC-reflection MCP** — derive MCP tools from *any* gRPC service via server
  reflection, not just go-micro-native handlers. Point the gateway at an external
  gRPC service and its methods become agent tools — a large jump in what an agent
  can operate.
- **Kubernetes operator + CRDs** — `Agent`, `Service`, and `Flow` as first-class
  Kubernetes resources; an operator reconciles them into Deployments wired to the
  registry. The production deployment story for teams already on K8s.

## Later — exploratory

- **Runtime-fitness loop** — a persistently-running dogfood app (Mu) plus an
  operator/canary loop role, so the autonomous loop evolves go-micro against
  **real runtime signal** (latency, errors, cost) with canary + rollback — not
  just green CI. The demand signal the loop is missing today.
- **HTTP/3 transport**; richer A2A live-stream reconnection (`tasks/resubscribe`,
  `input-required` handoffs); memory management (summarization, retrieval/RAG).

## Ongoing — hardening & DX (background, not the headline)

Continuous but **capped** so it never crowds out capability: cross-provider
conformance, failure/resilience (timeouts, cancellation, retry/backoff), the
0→1 and 0→hero getting-started contract, streaming/observability coherence, and a
seamless CLI inner loop (scaffold → run → chat → inspect → deploy). Real, but
maintenance — the loop should spend the majority of its cycles on the capability above,
not here.

## How it's sustained

The framework is the product, funded by sponsorship from those who run it — not a
hosted service, enterprise tier, or venture funding. See
[the v6 story](https://go-micro.dev/blog/27).

## Contributing & feedback

Pick an item, open an issue to discuss the approach, and submit a PR. Or join the
[Discord](https://discord.gg/G8Gk5j3uXr). Include tests, run `make test` and
`make lint`.

## Version support

- **v6** — active development (current).
- **v5** — security fixes only.
- **v4 and earlier** — end of life.

Major versions (v5 → v6) carry breaking changes; minors are backward-compatible.
See the [v5 → v6 migration guide](https://go-micro.dev/docs/guides/migration/v5-to-v6).
