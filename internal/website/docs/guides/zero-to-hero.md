---
layout: default
---

# 0→hero reference path

The 0→hero path is the maintained, no-secret reference for the Go Micro
services → agents → workflows lifecycle. It ties the CLI inner loop and the
runtime harness together so a contributor can prove the framework still works as
one system, not as separate demos.

Use it when you want to answer: "Can I scaffold a service, run it locally, talk
to an agent, inspect durable work, and reach the deployment boundary without
cloud credentials?"

## What the contract covers

| Boundary | Contract | CI check |
| --- | --- | --- |
| Scaffold | `micro new` generates a runnable service with and without MCP support. | `go test ./cmd/micro/cli/new -run TestZeroToOne -count=1` |
| Run | `micro run` remains the local development entry point. | `go test ./cmd/micro -run TestZeroToHeroCLIBoundaries -count=1` |
| Chat | `micro chat` remains the interactive agent entry point. | `go test ./cmd/micro -run TestZeroToHeroCLIBoundaries -count=1` |
| Inspect | `micro inspect agent`, `micro inspect flow`, and `micro flow runs` remain discoverable for run history. | `go test ./cmd/micro -run TestZeroToHeroCLIBoundaries -count=1` |
| Deploy | `micro deploy --dry-run` resolves deploy targets without touching remote infrastructure. | `go test ./cmd/micro/cli/deploy -run TestDeployDryRun -count=1` |
| Runtime | Real services, agents, durable flows, store-backed history, delegation, and A2A run with only the model mocked. | `./internal/harness/zero-to-hero-ci/run.sh` and `make provider-conformance-mock` |

## Run the runnable example

From the repository root, start with the support-desk example when you want to see the full lifecycle in one terminal:

```sh
go run ./examples/support
```

It starts typed services, a support agent, an event-driven intake flow, and an approval gate with a deterministic mock model. Change one service method, agent prompt, or guardrail decision and run it again to learn the system by modifying a working path.

## Run the whole no-secret path

From the repository root:

```sh
make harness
```

That target runs the scaffold contract, the CLI boundary smoke tests, the
0→hero runtime harnesses, the event-driven agent-flow harness, and mock provider
conformance. It is intentionally deterministic: no provider key, cloud account,
SSH access, or remote service is required.

## Run focused checks while iterating

Use the smaller checks when you are working on one seam:

```sh
# Scaffold → run/call contract.
go test ./cmd/micro/cli/new -run TestZeroToOne -count=1

# CLI inner-loop commands: run, chat, inspect, flow runs, deploy --dry-run.
go test ./cmd/micro -run TestZeroToHeroCLIBoundaries -count=1
go test ./cmd/micro/cli/deploy -run TestDeployDryRun -count=1

# Durable services → agents → workflows reference scenarios.
./internal/harness/zero-to-hero-ci/run.sh

# Event-as-prompt agent flow.
go run ./internal/harness/agent-flow

# Cross-provider semantics with the deterministic mock provider.
make provider-conformance-mock
```

## Reference scenarios

- [`examples/support`](https://github.com/micro/go-micro/tree/master/examples/support)
  is the runnable support-desk story: customers, tickets, notify, a support
  agent, an intake flow, and an approval gate in one no-secret example.
- [`examples/agent-plan-delegate`](https://github.com/micro/go-micro/tree/master/examples/agent-plan-delegate)
  is the smallest runnable planning/delegation example for multiple agents.
- [`internal/harness/plan-delegate`](https://github.com/micro/go-micro/tree/master/internal/harness/plan-delegate)
  is the compact 0→hero scenario: real task and notify services, a conductor
  agent, a comms agent, plan persistence, delegation, and a workflow handoff.
- [`internal/harness/universe`](https://github.com/micro/go-micro/tree/master/internal/harness/universe)
  boots a larger mini-world: inventory, payment, order confirmation, a concierge
  agent, durable checkpoint/resume, agent run history, flow run history, and A2A
  reachability.
- [`internal/harness/agent-flow`](https://github.com/micro/go-micro/tree/master/internal/harness/agent-flow)
  shows the event-driven path where a `user.created` event prompts an agent to
  call services and complete onboarding.

Together these scenarios keep the North Star executable: services expose typed
capabilities, agents use those capabilities with memory and guardrails, and
workflows compose the work over time.

## Keeping the guide honest

If you change the CLI inner loop, durable flow APIs, agent run history, or the
provider/tool semantics, update this guide and the harness in the same PR. The
point of 0→hero is not a polished sample app that drifts from reality; it is a
CI-verifiable contract that the documented lifecycle still works.
