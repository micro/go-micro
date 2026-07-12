# Changelog

All notable changes to Go Micro are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/) and versions
follow [Semantic Versioning](https://semver.org/), matching the git tags and
[GitHub releases](https://github.com/micro/go-micro/releases) (`v6.MINOR.PATCH`).
Releases are cut automatically as the loop merges improvements — a **minor**
bump when new features land (`### Added`/`### Changed`), a **patch** when it's
fixes/docs only; major bumps stay a human decision. The `[Unreleased]` section
below is kept current between tags and rolled into the next version when it ships.

> Earlier `2026.0x` headings are historical calendar-style markers from before
> v6 tagging; they are kept for continuity and not reused.

---

## [Unreleased]

### Added
- **Gemini streaming support** — the Gemini provider now supports streaming model responses. (`ai/gemini/`)
- **Model retry jitter controls** — model retry behavior can now use jitter controls to reduce synchronized retry bursts. (`ai/`, `agent/`)
- **Compacted memory summaries** — agent memory now exposes compacted run summaries for easier inspection and recovery. (`agent/`)
- **CLI input resume for agent runs** — the CLI can resume agent runs that require additional user input. (`cmd/micro/`, `agent/`)
- **Kubernetes CRD foundation (alpha)** — opt-in `Agent`/`Service`/`Flow` CustomResourceDefinitions (group `micro.go-micro.dev`, `v1alpha1`) plus a dependency-light `deploy/kubernetes` package that maps a resource to a Deployment (and a Service when a port is set), wired to the go-micro registry. `Render` is a pure function — no controller-runtime/client-go — so the mapping is CI-testable without a cluster. (`deploy/kubernetes/`)

### Changed
- **Remote agent chat streaming** — `micro chat` now streams replies from remote agents instead of waiting for the full response. (`cmd/micro/`, `agent/`)

### Fixed
- **Provider failure inspection metadata** — provider failures recorded during agent runs now retain classification metadata for inspection. (`agent/`, `ai/`)

### Security
- **x402 spend-cap hardening** — the paying `Client` now refuses a 402 whose `maxAmountRequired` is not a positive integer (a swallowed parse error or negative amount previously bypassed the budget cap), and a new `Config.RequireSettlement` fails closed when a paid request is served by a verify-only facilitator that never captures funds. (`wrapper/x402/`)

---

## [6.7.0] - July 2026

### Added
- **A2A streaming conformance harness** — A2A streaming behavior is now covered by focused conformance checks. (`gateway/a2a/`, `internal/harness/`)
- **Agent x402 spend budget guardrail** — agents now have spend budget guardrails for x402-paid tool calls. (`agent/`, `gateway/`)
- **First-agent chat/inspect fixture** — the maintained first-agent CLI fixture now covers chat and inspect boundaries together. (`internal/harness/`, `cmd/micro/`)
- **Zero-to-hero inspect transcript check** — the 0→hero harness now verifies the inspect transcript path stays visible in the lifecycle walkthrough. (`internal/harness/zero-to-hero-ci/`, `internal/website/docs/`)

### Changed
- **Agent stream run context propagation** — agent streams now preserve run context through streaming paths for more complete tracing and inspection. (`agent/`)
- **Postgres store pgx v5 migration** — the Postgres store now uses pgx v5. (`store/postgres/`, `go.mod`)
- **Plan-delegate plan persistence** — plan/delegate runs now persist plan state more defensively across harness scenarios. (`agent/`, `internal/harness/`)

### Fixed
- **Nested tool-call markup rejection** — agent argument parsing now rejects nested tool-call markup instead of accepting ambiguous tool input. (`agent/`)
- **Retry cancellation during backoff** — retry backoff now respects cancellation more reliably. (`agent/`, `ai/`)
- **Plan-delegate mock recovery regression gate** — the harness now catches plan/delegate mock recovery regressions before they ship. (`internal/harness/`, `agent/`)
- **First-agent fixture registration wait** — first-agent fixture registration is less race-prone during harness runs. (`internal/harness/`)
- **Memory stream Nack ordering** — memory stream Nack handling now preserves ordering more reliably. (`broker/memory/`)
- **Zero-to-hero fixture output race** — 0→hero fixture output is less race-prone during harness runs. (`internal/harness/zero-to-hero-ci/`)

### Documentation
- **First-agent quickcheck wayfinding** — public docs now keep the quickcheck path discoverable from the first-agent route. (`README.md`, `internal/website/docs/`)
- **Ordered 0→hero transcript** — docs and harness checks now keep the 0→hero transcript order explicit. (`internal/website/docs/`, `internal/harness/`)
- **First-agent debug breadcrumbs** — docs now surface the first-agent debug smoke path more clearly. (`internal/website/docs/`)
- **README badge cleanup** — the README no longer shows the Go Report Card badge. (`README.md`)

---

## [6.6.0] - July 2026

### Added
- **First-agent guide chain contract** — the harness now verifies the install → demo → examples → 0→hero guide chain stays connected for new agent builders. (`internal/harness/`, `internal/website/docs/`)
- **First-agent docs wayfinding guard** — the local harness now includes a focused no-network check for first-agent and 0→hero docs links. (`Makefile`, `internal/harness/`)
- **First-agent quickcheck breadcrumbs** — first-agent docs now surface quickcheck wayfinding for install, scaffold, chat, inspect, and recovery paths. (`internal/website/docs/`, `README.md`)
- **First-agent chat wayfinding verification** — the harness now verifies first-agent chat wayfinding remains discoverable from the public docs route. (`internal/harness/`, `internal/website/docs/`)

### Changed
- **Universe A2A reachability probe** — the universe harness now exercises A2A reachability more defensively. (`internal/harness/`)
- **AtlasCloud workspace repair fallback** — AtlasCloud fallback handling now recovers workspace-repair tool calls more reliably. (`ai/atlascloud/`, `agent/`)
- **AtlasCloud empty-argument tool repair** — AtlasCloud text tool-call repair now handles empty-argument calls more consistently. (`ai/atlascloud/`, `agent/`)

### Removed
- **`go-micro.dev/v6/ai/flow`** — the alias-only backward-compatibility shim is removed; import the canonical [`go-micro.dev/v6/flow`](flow) instead (same types and functions). It had no internal callers. (`ai/flow/`)

### Fixed
- **A2A fallback artifact text** — A2A fallback responses now avoid leaking provider artifact text into agent-visible output. (`gateway/a2a/`, `agent/`)
- **Launch readiness notification replays** — launch-readiness notification replay paths now deduplicate repeated side effects. (`agent/`, `internal/harness/`)
- **Plan-delegate harness cleanup** — plan/delegate harness cleanup is more reliable after conformance runs. (`internal/harness/`)
- **AtlasCloud spoken notify replays** — AtlasCloud fallback handling now collapses spoken notification replays more consistently. (`ai/atlascloud/`, `agent/`)
- **Agent-flow onboarding side effects** — onboarding side-effect checks are more stable across the agent-flow harness. (`agent/`, `internal/harness/`)
- **Plan-delegate plan-only side effects** — plan/delegate recovery now preserves plan-only side effects more reliably. (`agent/`, `internal/harness/`)
- **Checkpointed tool result recording** — checkpoint resume paths now guard tool-result recording against duplicate or stale writes. (`agent/`)
- **Agent timeout notification completion** — universe runs now finalize observed notifications more reliably after agent timeouts. (`agent/`, `internal/harness/`)
- **Completed plan-delegate side effects** — completed plan/delegate side effects are accepted more consistently in recovery paths. (`agent/`, `internal/harness/`)
- **Agent-flow onboarding notifications** — agent-flow onboarding notification recovery is more reliable across replay scenarios. (`agent/`, `internal/harness/`)

### Documentation
- **Agent-agnostic mention model** — loop docs now describe the mention-driven agent model without binding it to one coding agent. (`internal/docs/`, `.github/loop/`)
- **First-agent quickcheck docs** — public docs now surface the first-agent quickcheck path for faster troubleshooting. (`internal/website/docs/`)
- **Agent resume breadcrumbs** — docs now add clearer resume breadcrumbs for checkpointed agent runs. (`internal/website/docs/`)

### Security
- **Govulncheck vulnerability gate** — CI now includes a govulncheck gate and wires vulnerability failures into loop triage. (`.github/workflows/`, `cmd/micro/loop/`)
- **Dependency vulnerability patches** — toolchain and dependency updates patch reachable CVEs across the project. (`go.mod`, `go.sum`)

---

## [6.5.0] - July 2026

### Added
- **Agent stream provider conformance** — provider conformance now covers agent streaming behavior so streaming-capable providers stay aligned with the harness contract. (`agent/`, `internal/harness/`)
- **First-agent docs CLI parity check** — the harness now verifies first-agent docs commands match the CLI wayfinding surface. (`internal/harness/`, `internal/website/docs/`)
- **Focused CLI inner-loop contract** — the local harness now covers scaffold, run/chat/inspect, and deploy dry-run boundaries in one first-run contract. (`internal/harness/`)
- **First-agent wayfinding breadcrumbs** — first-agent docs and examples now have locked breadcrumb coverage from the README through the runnable examples. (`README.md`, `internal/website/docs/`, `examples/`)
- **Offline `micro new` contract** — project scaffolding now has an offline contract so the first service path stays runnable without network access. (`cmd/micro/`, `internal/harness/`)

### Changed
- **Provider model call timeouts** — model call timeout enforcement now wraps provider calls more defensively, reducing hangs in agent and harness paths. (`agent/`, `ai/`)
- **First-agent harness diagnostics** — getting-started harness logs now make first-run and 0→hero failures easier to locate. (`internal/harness/`)
- **MiniMax streaming conformance** — MiniMax streaming coverage now exercises broader provider conformance behavior. (`ai/minimax/`, `internal/harness/`)
- **AtlasCloud streaming tool capability** — AtlasCloud tool-streaming capability detection is now aligned with provider fallback behavior. (`ai/atlascloud/`, `agent/`)

### Fixed
- **Partial text tool calls** — text tool-call recovery now repairs partial function-style calls more reliably before fallback parsing continues. (`agent/`)
- **Retry timeout test stability** — retry timeout coverage is less race-prone. (`agent/`)
- **Checkpointed tool-call resume** — resumed agent runs now preserve checkpointed tool calls across startup resume paths. (`agent/`)
- **Model retry backoff contracts** — retry backoff behavior now has focused contract coverage for model-call failures. (`agent/`, `ai/`)
- **AtlasCloud conformance markers** — AtlasCloud fallback paths now preserve conformance markers through tool-call recovery. (`ai/atlascloud/`, `agent/`)
- **AtlasCloud delegate text fallback** — delegate text fallback recovery is more reliable for AtlasCloud responses. (`ai/atlascloud/`, `agent/`)
- **AtlasCloud incomplete plan repairs** — incomplete plan repair paths now recover more consistently in AtlasCloud fallback handling. (`ai/atlascloud/`, `agent/`)
- **AtlasCloud partial text tool calls** — AtlasCloud fallback handling now repairs partial text-rendered tool calls more reliably. (`ai/atlascloud/`, `agent/`)

### Documentation
- **Roadmap agent status** — public roadmap docs now reflect the current agent lifecycle status more consistently. (`internal/website/docs/`)
- **Agent resume limits** — docs now describe checkpoint resume boundaries for agent runs. (`internal/website/docs/`)
- **Zero-to-hero harness boundaries** — docs now clarify which 0→hero lifecycle checks are maintained by the local harness. (`internal/website/docs/`, `internal/harness/`)
- **First-agent wayfinding guard** — first-agent docs wayfinding now has tighter guard coverage around the README, docs, and examples chain. (`README.md`, `internal/website/docs/`)

---

## [6.4.0] - July 2026

### Added
- **Provider HTTP retry signals** — provider failures now preserve HTTP status and `Retry-After` details so retry classification and backoff can respond to rate limits and unavailable providers. (`ai/`)
- **Zero-to-hero deploy dry-run verification** — the maintained 0→hero harness now covers deploy dry-run boundaries for the services → agents → workflows lifecycle. (`internal/harness/`)
- **First-agent CLI wayfinding verification** — the harness now checks that first-agent CLI wayfinding stays discoverable. (`internal/harness/`)
- **Agent startup resume verification** — agent startup resume now has focused checkpoint coverage. (`agent/`, `internal/harness/`)
- **Direct first-agent chat prompts** — first-agent flows can accept direct chat prompts, reducing friction in the first useful conversation. (`cmd/micro/`, `agent/`)
- **Workflow run info on tool spans** — agent tool spans now include workflow run details for easier trace correlation. (`agent/`, `flow/`)

### Fixed
- **Stream fallback memory** — unsupported streaming attempts no longer leave stale duplicate user turns before fallback paths continue with non-streaming agent calls. (`agent/`)
- **Function-style text tool calls** — agent fallback parsing now recognizes provider replies that render tools as function-style calls, including nested JSON arguments. (`agent/`)
- **Plan/delegate notify recovery** — plan-delegate recovery now waits for recovered notify side effects and routes retries through the communications agent that owns the notification. (`internal/harness/`)
- **Onboarding side-effect enforcement** — the agent-flow harness now fails when required onboarding side effects are missing, making lifecycle regressions visible. (`internal/harness/`)
- **Plan/delegate notify stability** — notify recovery is more deterministic across retry and replay paths. (`agent/`, `internal/harness/`)
- **AtlasCloud MiniMax tool fallback** — AtlasCloud MiniMax service-tool fallback now handles 400 responses and follow-up retries more reliably. (`ai/atlascloud/`, `agent/`)

### Documentation
- **First-agent docs wayfinding guard** — the local harness now includes a focused no-network check for first-agent and 0→hero docs links. (`Makefile`, `internal/harness/`)

---

## [6.3.18] - July 2026

### Added
- **StreamAsk close cancellation** — agent streaming calls now cancel promptly when their runner closes, avoiding orphaned stream work. (`agent/`)
- **Agent resume pending helper** — agent durability now has a focused helper for resuming pending checkpointed runs. (`agent/`)
- **Agent tool retry tracing** — agent traces now include tool retry attempts for easier debugging of retry/fallback behavior. (`agent/`)
- **Shared-broker universe harness** — the universe harness now runs against the shared broker path, improving coverage of the same runtime wiring used by services, agents, and workflows. (`internal/harness/`)

### Fixed
- **Plan/delegate retry idempotency** — agent retries now preserve side-effect and notification dedupe across conformance retry paths, including completion and owner-notification edge cases. (`agent/`, `internal/harness/`)
- **AtlasCloud text tool calls** — AtlasCloud fallback handling now recovers more text-rendered tool calls from OpenAI-compatible responses. (`ai/atlascloud/`, `agent/`)
- **OpenAI-compatible text tool calls** — OpenAI-compatible providers now recover text-rendered tool calls more reliably. (`agent/`)
- **AtlasCloud multi-step follow-ups** — AtlasCloud tool fallback handling now continues multi-step tool follow-up paths more reliably. (`ai/atlascloud/`, `agent/`)

### Documentation
- **Agent debugging quickcheck** — docs now include a focused quickcheck path for first-agent debugging. (`internal/website/docs/`)
- **Website first-agent examples map** — website docs now link the maintained examples wayfinding map for the first-agent route. (`internal/website/docs/`)
- **Examples wayfinding index** — examples docs now provide a central map for first-agent, support, and interop examples. (`examples/`, `internal/website/docs/`)

---

## [6.3.17] - July 2026

### Added
- **First-agent examples CLI wayfinding** — `micro examples` now prints the maintained provider-free first-agent examples in copy/paste order. (`cmd/micro/`)
- **0→hero CLI entrypoint** — `micro zero-to-hero` now points developers at the maintained no-secret services → agents → workflows harness and runnable examples. (`cmd/micro/`)
- **First-agent tutorial smoke harness** — the first-agent tutorial path now has smoke coverage to keep the no-secret on-ramp runnable. (`internal/harness/`)
- **No-secret agent debugging smoke** — the no-secret agent debugging path now has smoke coverage for the first-agent troubleshooting flow. (`internal/harness/`)
- **Durable checkpoint resume smoke coverage** — durable agent resume after checkpointing now has focused smoke coverage. (`agent/`, `internal/harness/`)

### Fixed
- **Plan/delegate notify replays** — duplicate and replayed plan-delegate notifications are now idempotent, so resumed runs do not duplicate completed notifications. (`agent/`, `internal/harness/`)
- **Provider conformance scheduling** — provider conformance workflow dispatches now guard their scheduling path more reliably. (`.github/workflows/`)
- **Plan/delegate notification completion** — delegated notifications now preserve plan completion state more reliably, including duplicate, paraphrased, and delegated-owner notification paths. (`agent/`, `internal/harness/`)
- **AtlasCloud tool fallback** — AtlasCloud built-in tool schemas and follow-up tool fallback handling now recover conformance delegate retries more reliably. (`ai/atlascloud/`, `agent/`)
- **Agent conformance retry completion** — conformance retry prompts and completion handling are more deterministic for delegated agent runs. (`agent/`, `internal/harness/`)

### Documentation
- **First-agent quickstart numbering** — the first-agent on-ramp numbering is consistent across the README and website docs. (`README.md`, `internal/website/docs/`)
- **First-agent inspect command** — docs now use the maintained `micro inspect agent <name>` form. (`README.md`, `internal/website/docs/`)
- **`micro loop` quickstart wayfinding** — docs now surface the loop quickstart from the public docs index and README wayfinding. (`README.md`, `internal/website/docs/`)

---

## [6.3.16] - July 2026

### Added
- **No-secret agent demo CLI** — the CLI now surfaces `micro agent demo`, making the provider-free first-agent path discoverable from the installed binary. (`cmd/micro/`)
- **First-agent recovery doctor** — first-agent recovery checks now help diagnose install, scaffold, and provider setup issues before the live agent run. (`cmd/micro/`, `internal/website/docs/guides/`)

### Changed
- **Architecture lifecycle docs** — the architecture guide now leads with the services → agents → workflows lifecycle and the first-agent on-ramp. (`internal/website/docs/architecture.md`)
- **First-agent on-ramp** — README and website docs now lead new users through install troubleshooting, no-secret demos, the smallest first-agent example, debugging, and the 0→hero reference path in the same order. (`README.md`, `internal/website/docs/`)

### Fixed
- **Config close idempotency** — config close paths now tolerate repeated closes safely. (`config/`)
- **OpenTelemetry child span events** — agent traces now preserve child span events more reliably. (`agent/`)

### Documentation
- **Security reporting** — security docs now route vulnerability reports through GitHub Security Advisories. (`SECURITY.md`, `internal/website/docs/`)
- **Install troubleshooting** — the first-agent on-ramp now includes clearer install and PATH recovery guidance. (`internal/website/docs/guides/install-troubleshooting.md`)

---

## [6.3.15] - July 2026

### Added
- **Anthropic streaming** — the Anthropic provider now supports Messages SSE streaming and is registered as a streaming-capable provider, with capability docs and parser coverage. (`ai/anthropic/`, `internal/website/docs/guides/`)
- **AP2 mandate foundation for A2A** — the A2A gateway now has the shared payment-mandate foundation needed for AP2-style agent payment flows. (`gateway/a2a/`)
- **Smallest first-agent example** — a no-secret, mock-model first-agent example gives the on-ramp a minimal runnable starting point. (`examples/first-agent/`)

### Changed
- **First-agent CLI next steps** — CLI output now points new users toward the maintained first-agent path after scaffold/run milestones. (`cmd/micro/`)

### Fixed
- **Plan/delegate completion** — plan-delegate runs now preserve completed steps, guard ordering, require notify-before-completion, and stabilize checkpoint continuation paths. (`agent/`, `internal/harness/`)
- **Provider text tool calls** — AtlasCloud and weaker-model fallback paths now recover tagged, `Create`-suffixed, mixed text/tool-call, and follow-up tool calls more reliably. (`agent/`, `ai/atlascloud/`)
- **First-agent broker isolation** — the first-agent harness now isolates broker state more reliably across runs. (`internal/harness/`)

### Documentation
- **First-agent example path** — docs and website wayfinding now surface the smallest example, no-secret transcript, and 0→hero path together. (`README.md`, `internal/website/docs/`)
- **Agent operations guidance** — agent debugging docs now include operational failure guidance, inspect hints, and durable resume pointers. (`internal/website/docs/guides/`)

---

## [6.3.14] - July 2026

### Added
- **MiniMax provider** — run agents against MiniMax's `MiniMax-M3` model via its OpenAI-compatible endpoint, with tool calling and streaming; auto-detected from the base URL. (`ai/minimax/`)
- **`micro loop` security role** — a new opt-in loop role (`--roles …,security`) that periodically audits a repo for vulnerabilities and files `security` issues. It is deliberately conservative: it never auto-merges fixes and never publishes exploit detail in public issues (responsible disclosure), and risky fixes are marked `needs-human`. go-micro now runs it against its own attack surface (MCP/A2A gateways, x402, auth, provider URLs, agent tool loop, deps). (`cmd/micro/loop/`)
- **Agent run tracing** — agent model streaming and run-event kinds now emit richer trace detail for debugging agent execution. (`agent/`)

### Changed
- **Agent memory** — streamed agent replies are persisted in conversation memory so later turns can reference streamed responses. (`agent/`)

### Fixed
- **Plan/delegate completion** — agents now continue unfinished plan steps more reliably, fail checkpointed runs that leave delegated plans unfinished, recover from unknown plan-delegate tool calls, avoid duplicate side effects, and complete timeout paths deterministically. (`agent/`)
- **AtlasCloud tool calls** — streaming and request fallback handling now recovers tool-call results from provider responses that omit the expected structured fields. (`ai/atlascloud/`)
- **Agent preflight diagnostics** — provider setup failures now surface more actionable errors before an agent run starts. (`agent/`)
- **A2A fallback streams** — fallback stream validation is stricter for malformed or incomplete A2A streaming responses. (`gateway/a2a/`)
- **File-store test isolation** — file-store expiry and table tests are less timing-sensitive and isolate their state more reliably. (`store/file/`)

### Documentation
- **First-agent debugging path** — docs now include no-secret transcript checkpoints, durable resume examples, and clearer CLI/website wayfinding for first-agent debugging. (`README.md`, `internal/website/docs/`, `examples/agent-durable/`)

---

## [6.3.13] - July 2026

### Added
- **`micro loop`** — scaffold an autonomous improvement loop into any repository: GitHub Actions workflows dispatched to an @mention-driven coding agent, across up to five roles — `planner` (ranked queue), `builder` (top item as a single-concern PR, auto-merged on green CI), `triage` (CI failures → fix issues), and opt-in `coherence` (docs/CHANGELOG alignment) and `release` (daily patch tag). Each dispatch role's instruction lives in an editable `.github/loop/prompts/<role>.md` file — the workflow is the mechanism, the prompt is the policy — so a repo customizes behavior without forking the CLI. `micro loop init --roles …` writes it all; `micro loop verify` checks the wiring. This is the loop that maintains go-micro itself, generalized. (`cmd/micro/loop/`)

### Changed
- **x402 payments** — settlement now covers CDP facilitator authentication and conformance edge cases. (`wrapper/x402/`)

### Fixed
- **Plan/delegate harnessing** — side effects and notifications are now idempotent and deterministic across duplicate, alias, order-scoped, and reachability scenarios. (`agent/`, `internal/harness/`)

### Documentation
- **First-agent on-ramp** — quickstart docs now connect the no-secret first-agent transcript, example map, and 0→hero path. (`README.md`, `internal/website/docs/`)
- **Ollama provider docs** — the provider surface, capability matrix, and examples now document local and cloud behavior. (`internal/website/docs/`, `examples/agent-ollama/`)

---

## [6.3.12] - July 2026

### Added
- **Ollama provider** — run agents against open-weight models locally (`/api/chat`, NDJSON streaming) or via Ollama Cloud (OpenAI-compatible `/v1/chat/completions`, SSE), auto-detected from the base URL, with tool calling in both modes. Point any agent at a non-default endpoint with the new `agent.BaseURL` / `micro.AgentBaseURL` option. (`ai/ollama/`, `examples/agent-ollama/`)
- **Retrieval-backed agent memory** — agents can recall relevant prior turns by similarity, not just the recent window, with a summarizer hook that compacts older history so long conversations stay in budget. (`agent/`)
- **Scheduled flows** — a flow can run an agent (or any step) on a cron-style schedule, with the dispatch traced end to end. (`flow/`)
- **Flow verification/grader loop** — a workflow can grade its own step output against a rubric and retry until it passes, plus run-trace analysis to surface where a flow spends its time. (`flow/`)
- **A2A streaming & continuity** — outbound agent streaming flows through the A2A binding (`message/stream`), with `tasks/resubscribe` and `input-required` handoffs for multi-turn interop. (`gateway/a2a/`)

### Changed
- **Agent tool-call resilience** — opt-in retries around agent tool calls, and a fallback that executes tool calls emitted as text by weaker models so they still make progress. (`agent/`)
- **Hardened agent durability** — terminal failure statuses are classified and surfaced, and durable resume-after-restart is covered by tests. (`agent/`)

### Documentation
- **"Your first agent" walkthrough** and a canonical 0-to-hero reference path, lowering the on-ramp from install to a running agent. (`internal/website/docs/`)
- **Discord** linked prominently across the README, website nav/footer, and docs. (`https://discord.gg/G8Gk5j3uXr`)

---

## [6.0.0] - June 2026

The AI-native major release. Breaking changes are listed first; everything
else is additive. See the [v5 → v6 migration guide](internal/website/docs/guides/migration/v5-to-v6.md) — it's a small upgrade.

### Changed (breaking)
- **Module path is now `go-micro.dev/v6`.** Update imports (`go-micro.dev/v5/...` → `go-micro.dev/v6/...`) and `go install go-micro.dev/v6/cmd/micro@v6`.
- **TLS verification is on by default.** v5 skipped verification unless `MICRO_TLS_SECURE=true`; v6 verifies by default. `MICRO_TLS_SECURE` is removed — set `MICRO_TLS_INSECURE=true` (or call `tls.InsecureConfig()`) for self-signed/dev certs.
- **`micro.NewService(name, opts...)` is the service constructor**, symmetric with `NewAgent`/`NewFlow`. `micro.New(name, opts...)` remains as a deprecated alias; the old name-less `micro.NewService(opts...)` form is removed (pass the name positionally). Generators emit the new form.
- **JWT auth ported in-module.** The external `github.com/micro/plugins/v5/auth/jwt` (pinned to v5) is replaced by `go-micro.dev/v6/auth/jwt/token`, now on the maintained `golang-jwt/jwt/v5`; the deprecated `dgrijalva/jwt-go` dependency is dropped.

### Added
- **A2A protocol — both directions** — `gateway/a2a` exposes registered agents over the open Agent2Agent (A2A) protocol so agents on other frameworks can discover and call them: Agent Cards are generated from registry metadata (the same way the MCP gateway derives tools), and incoming tasks are translated to the agent's existing `Agent.Chat` RPC, with no per-agent code (`micro a2a serve`). The outbound `a2a.Client` calls external A2A agents by URL, wired into `flow.A2A(url)` (a workflow step) and `delegate` to an `http(s)` URL (from inside an agent). An agent can also serve A2A **directly** without a gateway via `AgentA2A(addr)` (`a2a.NewAgentHandler`), handling tasks in-process. The JSON-RPC binding includes `message/send`, `message/stream` (SSE), `tasks/get`, multi-turn continuation by `taskId`/`contextId`, best-effort push notification callbacks, `tasks/resubscribe`, `input-required` handoffs, and card discovery. (`gateway/a2a/`, `cmd/micro/a2a/`)
- **Agents (`micro.NewAgent`)** — an agent is a service with an LLM inside: it discovers its assigned services as tools, runs the model's tool loop, registers a `Chat` RPC endpoint, and is reachable like any service. `Ask` for programmatic use; `micro chat` discovers and routes to agents; `micro agent list`/`describe`. (`agent/`)
- **Plan & delegate** — two built-in agent tools added to every agent: `plan` (an ordered, store-persisted plan surfaced back in the prompt) and `delegate` (hand a self-contained subtask to a registered agent over RPC, otherwise to an ephemeral sub-agent). No harness or graph — they're plain tools. (`agent/builtin.go`, `examples/agent-plan-delegate/`)
- **Agent guardrails** — `MaxSteps` (stop on count), `LoopLimit` (stop repeated no-progress calls; on by default), and `ApproveTool` (human-in-the-loop / policy gate before each action), enforced at the one point every tool call passes through. (`agent/`, guide + blog)
- **Pluggable agent memory & custom tools** — durable store-backed conversation memory by default, swappable via `AgentMemory`; register any function as a tool with `AgentTool`.
- **Workflows (`micro.NewFlow`)** — event-driven orchestration that maps to Anthropic's workflow/agent split: an event triggers a deterministic step (or ordered durable steps), or dispatches to an agent with `FlowAgent`. (`flow/`)
- **Flow loops (`FlowLoop`)** — a flow step that runs a body step repeatedly, carrying state across passes, until a stop condition is met or a hard iteration cap is hit. Stop on a code-defined predicate (`FlowUntil`) or let the model judge it done (`FlowUntilLLM` — the supervised "Ralph" loop); `FlowLoopMax` is the guardrail that guarantees termination, and `FlowOnIteration` reports progress. (`flow/loop.go`, `examples/flow-loop/`, guide)
- **x402 payments** — opt-in per-call payments for tools via the x402 standard, with a pluggable facilitator and a consumer-side client + budget; the MCP gateway can advertise and require payment per tool. (`wrapper/x402/`, guide + blog)
- **Scoped store state** — `store.Scope(s, database, table)` returns a store handle that confines every operation to a database/table without mutating the shared store (unlike `Init(Table(...))`, which is process-global and races between co-located components). Services, agents, and flows now each keep their state in their own table (`service/{name}`, `agent/{name}`, `flow/{name}`); the service path replaces the old `Init(store.Table(name))` global mutation with a scoped handle.
- **Flow discovery & history CLI** — running flows now register in the registry as `type=flow` (and deregister on `Stop`), so they're discoverable like agents: `micro flow list` shows running flows, `micro flow runs <name>` shows a flow's durable run history from the store, and `micro agent history <name>` shows an agent's stored conversation. Live state comes from the registry; durable history from the scoped store.
- **Durable workflows** — a flow can now be an ordered list of steps (a task with stages) that is checkpointed before and after each step, so a run survives a crash and resumes where it stopped without re-running completed steps. State carries a typed payload plus a `Stage` marker; flow-level `Retry` with a per-step override; runs retained for audit unless `DeleteOnSuccess`. Step actions: `Call` (RPC), `LLM` (model turn), `Dispatch` (to an agent), or any `StepFunc`. Durability is a pluggable `Checkpoint` (store-backed by default; implement the interface for Temporal/Restate). Runnable example: `examples/flow-durable/`. Blog: "Durable Workflows" (`internal/website/blog/24.md`).
- **Agent tool-execution wrappers** — `AgentWrapTool` registers middleware around an agent's tool calls, the tool-side analogue of `client.CallWrapper`/`server.HandlerWrapper`. Use it for logging, metrics, retries, or policy; wrappers compose outermost-first and run outside the built-in guardrails. Includes a runnable example with observe + retry wrappers (`examples/agent-wrap-tool/`).
- **Agent platform showcase** — full platform example (Users, Posts, Comments, Mail) mirroring [micro/blog](https://github.com/micro/blog), demonstrating how existing microservices become agent-accessible with zero code changes (`examples/mcp/platform/`).
- **Blog post: "Your Microservices Are Already an AI Platform"** — walkthrough of agent-service interaction patterns using real-world services (`internal/website/blog/7.md`).
- **Circuit breakers for MCP gateway** — per-tool circuit breakers protect downstream services from cascading failures. Configurable max failures, open-state timeout, and half-open probing. Available via `Options.CircuitBreaker` and `--circuit-breaker` CLI flag (`gateway/mcp/circuitbreaker.go`).
- **Helm chart for MCP gateway** — official Helm chart at `deploy/helm/mcp-gateway/` with Deployment, Service, ServiceAccount, HPA, and Ingress templates. Supports Consul/etcd/mDNS registries, JWT auth, rate limiting, audit logging, per-tool scopes, TLS ingress, and auto-scaling.
- **MCP gateway benchmarks** — comprehensive benchmark suite for tool listing, lookup, auth, rate limiting, and JSON serialization (`gateway/mcp/benchmark_test.go`)
- **Workflow example** — cross-service orchestration demo with Inventory, Orders, and Notifications services showing agents chaining multi-step workflows from natural language (`examples/mcp/workflow/`)
- **Docker Compose deployment** — production-like setup with Consul registry, standalone MCP gateway, and Jaeger tracing in one `docker-compose up` (`examples/deployment/`)

---

## [2026.03] - March 2026

### Added

#### Developer Experience
- **`micro new` MCP templates** — `micro new myservice` generates MCP-enabled services with doc comments, `@example` tags, and `WithMCP()` wired in. Use `--no-mcp` to opt out.
- **`micro.NewService("name")` unified API** — single way to create services: `micro.NewService("greeter")` or `micro.NewService("greeter", micro.Address(":8080"))`. Replaces `micro.NewService()` + `service.New()` dual API.
- **`service.Handle()` simplified registration** — register handlers with `service.Handle(new(Greeter))` instead of manual `server.NewHandler` + `server.Handle`.
- **`micro.NewGroup()` modular monoliths** — run multiple services in one binary with shared lifecycle: `micro.NewGroup(users, orders).Run()`.
- **`mcp.WithMCP()` one-liner** — add MCP to any service with a single option: `micro.NewService("name", mcp.WithMCP(":3001"))`.
- **CRUD example** — contact book service with 6 operations, rich agent docs, and validation patterns (`examples/mcp/crud/`).

#### MCP Gateway
- **WebSocket transport** — bidirectional JSON-RPC 2.0 streaming over WebSocket for real-time agent communication (`gateway/mcp/websocket.go`).
- **OpenTelemetry integration** — full span instrumentation across HTTP, stdio, and WebSocket transports with W3C trace context propagation (`gateway/mcp/otel.go`).
- **Standalone gateway binary** — `micro-mcp-gateway` with Docker support for running the MCP gateway independently of services.
- **Per-tool auth scopes** — service-level (`server.WithEndpointScopes()`) and gateway-level (`Options.Scopes`) scope enforcement with bearer token auth.
- **Rate limiting** — per-tool token bucket rate limiting (`Options.RateLimit`).
- **Audit logging** — immutable audit records per tool call with trace ID, account, scopes, duration, and errors (`Options.AuditFunc`).

#### AI Model Package
- **`model.Model` interface** — unified AI provider abstraction with `Generate()` and `Stream()` methods.
- **Anthropic Claude provider** — `model/anthropic` with tool execution and auto-calling.
- **OpenAI GPT provider** — `model/openai` with provider auto-detection from base URL.

#### Agent SDKs
- **LangChain SDK** — `contrib/langchain-go-micro/` Python package with auto-discovery, tool generation, and multi-agent workflow examples.
- **LlamaIndex SDK** — `contrib/go-micro-llamaindex/` Python package with RAG integration examples.

#### Documentation
- **AI-native services guide** — building services for AI agents from scratch
- **MCP security guide** — auth, scopes, and audit logging
- **Tool descriptions guide** — writing doc comments that improve agent performance
- **Agent patterns guide** — architecture patterns for agent integration
- **Error handling guide** — writing agent-friendly error responses with typed errors
- **Troubleshooting guide** — common MCP issues and solutions
- **Migration guide** — add MCP to existing services in 5 minutes

#### CLI
- **`micro mcp serve`** — start MCP server (stdio for Claude Code, HTTP for web agents)
- **`micro mcp list`** — list available tools (human-readable or JSON)
- **`micro mcp test`** — test tools with JSON input
- **`micro mcp docs`** — generate tool documentation
- **`micro mcp export`** — export to LangChain, OpenAPI, or JSON formats

#### Agent Playground
- **Chat-focused UI** — redesigned playground with collapsible tool calls, real-time status, and thinking indicators
- **Provider settings** — configurable OpenAI/Anthropic provider, model, and API key

### Changed
- Service interface moved to `service.Service` with `micro.Service` as a type alias for backward compatibility.
- `service.New()` returns `service.Service` interface (was `*ServiceImpl`).
- `service.NewGroup()` accepts `service.Service` interface (was `*ServiceImpl`).
- `go.mod` template in `micro new` updated to Go 1.22.

### Fixed
- Handler `Handle()` method accepts variadic `server.HandlerOption` for scopes and metadata.
- Store initialization uses service name as table automatically.
- Service `Stop()` properly aggregates errors from lifecycle hooks.

---

## [2026.02] - February 2026

### Added
- **MCP gateway library** — `gateway/mcp/` with HTTP/SSE and stdio transports, service discovery, tool generation, and JSON schema generation from Go types (2,500+ lines).
- **CLI integration** — `micro run --mcp-address` flag to start MCP alongside services.
- **Documentation extraction** — auto-extract tool descriptions from Go doc comments with `@example` tag and struct tag parsing.
- **Blog post** — "Making Microservices AI-Native with MCP"
- **MCP examples** — `examples/mcp/hello/` and `examples/mcp/documented/`

---

## [2026.01] - January 2026

### Added
- **`micro deploy`** — deploy services to any Linux server via SSH + systemd with `micro deploy user@server`.
- **`micro build`** — build Go binaries and Docker images with `micro build --docker`.
- **Blog post** — "Introducing micro deploy"

---

_For earlier changes, see the [git log](https://github.com/micro/go-micro/commits/master)._
