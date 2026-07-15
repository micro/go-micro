# Gap Audit — the integration/exposure surface (MCP, A2A, x402) + foundations

A code-grounded robustness audit of the surfaces the strategy rests on, plus the
two foundations a real app (Mu) discovered it needed. Pair this with the
"requirements discovered from Mu" notes — together they are the roadmap's
evidence base. Each finding is `file:line`, severity, and what "robust in
practice" requires.

## Headline

All four surfaces are **well-built internally and fragile at the edge.** The
strategic spine — **MCP gateway, A2A, x402** — is *demo-robust, not
production-robust*: it works go-micro-to-go-micro and is **unverified or broken
against the real external clients the whole "integration and exposure" strategy
depends on.** Not one test drives a real external MCP host, a real third-party
A2A SDK, or a real x402 facilitator/wallet.

There are two kinds of hardening, and the loop was doing the wrong one: guarding
docs and chasing a weak provider's quirks is grooming; making MCP actually speak
MCP to Claude Desktop, A2A interoperate with a real external agent, and x402
actually settle is **strategic** hardening. The axis is *advances the strategy*
vs *grooms a proxy*, not *capability* vs *hardening*.

## MCP gateway — `gateway/mcp/`
Works as go-micro plumbing; does not speak MCP to the outside world.

- **BLOCKER** `mcp.go:610` — default HTTP transport is bespoke `{tool,input}` REST, not JSON-RPC/MCP. A conformant JSON-RPC handler exists (`httpjsonrpc.go` `NewHandler`) but is **never mounted**. → mount it / implement Streamable HTTP; unify transports behind one pre-call pipeline.
- **BLOCKER** `stdio.go:327`, `websocket.go:306` — tool results are `fmt.Sprintf("%v", result)` → Go map-syntax, not JSON, on the path Claude Desktop uses. **Zero stdio tests.** → marshal JSON; add a stdio round-trip test.
- **BLOCKER** `stdio.go:297`, `websocket.go:278` — downstream errors returned as JSON-RPC protocol errors, not `{isError:true}` results. → wrap as tool-error results.
- **BLOCKER** `websocket.go:20` — `CheckOrigin` always `true` (DNS-rebinding); and `/mcp/ws` **bypasses payment + circuit breaker** → paid tools free over WS. → origin allowlist; one shared pre-call pipeline.
- **MAJOR** deregistered tools never pruned (`mcp.go:280`); watcher never recovers + no `list_changed` (`mcp.go:564`); unbounded goroutines, no `recover()` (`stdio.go:112`); no HTTP/WS timeouts or body limits; unauthenticated `micro_store_write`/`micro_broker_publish` by default (`mcp.go:474`).

## A2A gateway — `gateway/a2a/`
Clean binding; cross-framework interop unproven.

- **MAJOR** `a2a.go:111` — well-known path is `agent.json`; spec 0.3.0 serves `agent-card.json` → external clients 404. → serve both.
- **MAJOR** `a2a.go:587` — `message/stream` emits full `Task` snapshots, not `TaskStatusUpdateEvent`/`TaskArtifactUpdateEvent` with `final:true`; `:596` sets JSON-RPC `result`+`error` together (spec violation). → emit discriminated update events.
- **MAJOR** `a2a.go:584` — streaming ignores write errors / client disconnect (burns tokens on a dead socket); `:531` "streaming" is single-shot despite `streaming:true`; `:508` `tasks/cancel` is a stub; `:874` push callbacks SSRF-open + auth ignored; **no gateway auth / no security schemes** (`:342`); in-memory state breaks multi-replica (`:466`).
- **Critical:** no test against a real third-party A2A SDK — all interop claims self-certified.

## x402 — `wrapper/x402/`
Clean, spec-aware scaffold; no real money can move.

- **BLOCKER** — no real wallet `Payer` (no EIP-3009 signer); buyer `Client` wired into nothing — the agent's "spend budget" (`agent/builtin.go:380` `spendWrap`) is bookkeeping that never pays; CDP mainnet settlement unreachable from the CLI (creds never attached). This is the flagship (#4786).
- **MAJOR** `client.go:84` — `ParseInt` error swallowed → malformed amount parses to `0`, spend-cap check trivially passes while the Payer signs the string amount. **Fix as part of #4786.**
- **MAJOR** `x402.go:225` — verify-only facilitator serves the resource for free; `:198` no replay/idempotency; `:262` non-conformant settlement header; `client.go:83` no network/asset validation before signing.

## Foundations (Mu-discovered)

### In-process dispatch — `client/`, `transport/`
- **MAJOR** `client/rpc_client.go:148` — no in-process fast-path: an in-process `Call` still dials a transport and simulates a network hop; `transport/memory.go:82` double-serializes (gob over a pipe on top of the RPC codec) with ~4–5 goroutine handoffs. ~64µs/187 allocs confirmed. → a local transport / direct `router.ServeRequest` dispatch (the server already keeps a process-local handler table; the codec already passes `*Frame` bodies through unserialized). **Low-risk; plausibly low-single-digit µs.**

### Durable agentic workflow — `flow/`
`flow/` is genuinely close on the *deterministic* axis (checkpoints, resumes without replaying completed steps, `ParentID`, retry). The gap is the *agentic* axis:
- **BLOCKER** `flow/steps.go:107` — no human-in-the-loop pause (no `waiting` state / `Resume(runID, input)`). Exactly what Mu hand-rolled. → add a `waiting` status + await-input signal + resume-with-input.
- **MAJOR** `flow/steps.go:269` — the agent's dynamic plan→tool→tool loop is one opaque flow step; a crash mid-turn replays every tool call. The durable unit is a fixed step list, not the agent's pausable per-tool-call loop — the convergence thesis is unmet. → checkpoint per tool call.
- **MAJOR** `flow/loop.go:82` `Loop` not per-iteration checkpointed; `flow/steps.go:457` at-least-once, not exactly-once (duplicate side effects on resume); `:158` no run leasing for multi-replica.

## Build order (prioritized)

1. **MCP stdio: real JSON + `isError` results + a round-trip test.** Cheapest, highest-impact — the Claude Desktop path, currently emitting garbage. *(loop-buildable)*
2. **MCP: mount JSON-RPC as the HTTP transport + unify all transports behind one pre-call pipeline.** *(architectural — human-reviewed)*
3. **x402 flagship (#4786) done right:** signing `Payer`, buyer wired into the agent seam, budget-bypass fix, require a real `Settler`. *(mixed — the wiring is human-reviewed; the budget-bypass fix is loop-buildable)*
4. **Durable agentic workflow:** HITL pause + per-tool-call checkpointing in `flow`. *(architectural — human-reviewed)*
5. **In-process dispatch fast-path** (local transport). *(foundational, low-risk)*
6. **A2A external conformance:** well-known path + SSE event shapes, then a test against a real third-party A2A SDK. *(loop-buildable + a test-setup task)*

The architectural items (2, 4, 5) are the "real 1:1 development" work — too central and too ambiguous to hand to an autonomous agent. The well-scoped items (1, 3-budget-fix, 6) are what a forced, well-defined loop task looks like.
