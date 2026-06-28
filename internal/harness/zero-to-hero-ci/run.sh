#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

# Keep the developer inner-loop boundaries executable and discoverable in CI
# without secrets or long-running daemons.
go test ./cmd/micro -run TestZeroToHeroCLIBoundaries -count=1

# Deterministic no-secret reference scenarios. These use the real Go Micro
# runtime and mock only the LLM provider.
go test ./internal/harness/universe ./internal/harness/plan-delegate -run 'Test.*Harness|TestPlanDelegateEndToEnd|TestPlanDelegateFlowHandoff' -count=1
