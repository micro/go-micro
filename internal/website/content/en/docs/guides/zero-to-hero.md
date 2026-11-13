---
title: "0→hero reference path"
---

The 0→hero path is the maintained, no-secret reference for the Go Micro
services → agents lifecycle. It ties the CLI inner loop and the
runtime harness together so a contributor can prove the framework still works as
one system, not as separate demos.

Use it when you want to answer: "Can I scaffold a service, run it locally, talk
to an agent, inspect durable work, and reach the deployment boundary without
cloud credentials?"

## What the contract covers

| Boundary | Contract | CI check |
| --- | --- | --- |
| Scaffold | `micro new` generates a runnable service with and without MCP support. | `go test ./cmd/micro/cli/new -run TestZeroToOne -count=1` |
| First-agent wayfinding | README, website index/quickstart, examples, and no-secret/0→hero docs keep the no-secret → first-agent → debugging → 0→hero links present and in order. | `go test ./internal/harness/zero-to-hero-ci -run TestFirstAgentWayfinding -count=1` |
| First agent | `micro new`, `micro agent preflight`, `micro run`, `micro chat`, and `micro inspect agent <name>` stay available for the documented first-agent walkthrough. | `go test ./cmd/micro -run TestFirstAgentWalkthroughCLIBoundaries -count=1` |
| Run | `micro run` remains the local development entry point. | `go test ./cmd/micro -run TestZeroToHeroCLIBoundaries -count=1` |
| Chat | `micro chat` remains the interactive agent entry point. | `go test ./cmd/micro -run TestZeroToHeroCLIBoundaries -count=1` |
| Inspect | `micro inspect agent <name>` and `micro agent history <name>` expose agent run history without provider keys. | `go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentDebuggingSmoke -count=1` |
| Deploy | `micro deploy --dry-run prod` resolves the documented deploy target without touching remote infrastructure. | `go test ./internal/harness/zero-to-hero-ci -run TestZeroToHeroDeployDryRunCommandSmoke -count=1` |
| Smallest first agent | `examples/first-agent` runs one service-backed agent with a deterministic mock model and no provider key. | `go test ./examples/first-agent -run TestRunFirstAgent -count=1` |
| Runtime reference app | `examples/support` runs typed services, an agent using those services as tools, and an approval gate with only the model mocked. | `go test ./examples/support -run 'TestRunSupportMockSmoke|TestZeroToHeroReadmeDocumentsLifecycle|TestZeroToHeroInspectTranscript' -count=1` |
| Ordered 0→hero transcript | The maintained CI transcript walks scaffold → run/chat/inspect → support-agent chat → deploy dry-run without provider keys. | `make zero-to-hero-transcript` |
| Runtime harnesses | Real services, agents, store-backed history, delegation, and A2A run with only the model mocked. | `./internal/harness/zero-to-hero-ci/run.sh` and `make provider-conformance-mock` |

## Find the one-command entrypoint

After installing the CLI, ask `micro` for the maintained no-secret lifecycle command:

```sh
micro zero-to-hero
```

The command prints the exact harness command below plus the smaller runnable examples, so a new developer can discover the 0→hero path from CLI help instead of translating this guide by hand.

## Run the runnable example

From the repository root, start with the smallest service-backed agent when you want the fastest no-secret success path:

```sh
go run ./examples/first-agent
```

Then run the support-desk example when you want to see the full lifecycle in one terminal:

```sh
go run ./examples/support
```

It starts typed services, a support agent, and an approval gate with a deterministic mock model. Change one service method, agent prompt, or guardrail decision and run it again to learn the system by modifying a working path.

## Run the whole no-secret path

From the repository root:

```sh
make harness
```

For the focused ordered transcript only, run:

```sh
make zero-to-hero-transcript
```

That target runs the scaffold contract, the CLI boundary smoke tests, runtime harnesses, and mock provider
conformance. It is intentionally deterministic: no provider key, cloud account,
SSH access, or remote service is required.

## Run focused checks while iterating

Use the dedicated inner-loop target when you need the provider-free CLI contract in one focused command:

```sh
make inner-loop
```

Use the smaller checks when you are working on one seam:

```sh
# Install script and first-run CLI boundary, with no network or provider keys.
make install-smoke

# Scaffold → run/call contract.
go test ./cmd/micro/cli/new -run TestZeroToOne -count=1

# First-agent walkthrough boundary: scaffold, preflight, run, chat, inspect.
go test ./cmd/micro -run TestFirstAgentWalkthroughCLIBoundaries -count=1

# CLI inner-loop commands: run, chat, inspect, deploy --dry-run.
go test ./cmd/micro -run TestZeroToHeroCLIBoundaries -count=1
go test ./cmd/micro/cli/deploy -run TestDeployDryRun -count=1
go test ./internal/harness/zero-to-hero-ci -run TestZeroToHeroDeployDryRunCommandSmoke -count=1

# Smallest no-secret service-backed first agent.
go test ./examples/first-agent -run TestRunFirstAgent -count=1

# Maintained 0→hero support-desk reference app.
go test ./examples/support -run 'TestRunSupportMockSmoke|TestZeroToHeroReadmeDocumentsLifecycle|TestZeroToHeroInspectTranscript' -count=1

# Durable services → agents reference scenarios.
./internal/harness/zero-to-hero-ci/run.sh


# Cross-provider semantics with the deterministic mock provider.
make provider-conformance-mock
```

## Reference scenarios

- [`examples/first-agent`](https://github.com/micro/go-micro/tree/master/examples/first-agent)
  is the smallest no-secret service-backed agent: one notes service, one scoped
  assistant agent, and a deterministic mock model.
- [`examples/support`](https://github.com/micro/go-micro/tree/master/examples/support)
  is the runnable support-desk story: customers, tickets, notify, a support
  agent and an approval gate in one no-secret example.
- [`examples/agent-plan-delegate`](https://github.com/micro/go-micro/tree/master/examples/agent-plan-delegate)
  is the smallest runnable planning/delegation example for multiple agents.


Together these scenarios keep the North Star executable: services expose typed
capabilities, agents use those capabilities with memory and guardrails, and
agents can delegate to each other.

## Keeping the guide honest

If you change the CLI inner loop, agent run history, or the
provider/tool semantics, update this guide and the harness in the same PR. The
point of 0→hero is not a polished sample app that drifts from reality; it is a
CI-verifiable contract that the documented lifecycle still works.
