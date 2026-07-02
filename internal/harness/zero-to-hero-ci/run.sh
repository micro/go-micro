#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

# Keep the developer inner-loop boundaries executable and discoverable in CI
# without secrets or long-running daemons.
go test ./cmd/micro -run 'TestFirstAgentWalkthroughCLIBoundaries|TestZeroToHeroCLIBoundaries' -count=1
go test ./cmd/micro/cli/deploy -run TestDeployDryRun -count=1

# Deterministic no-secret reference scenarios. These use the real Go Micro
# runtime and mock only the LLM provider. The support example is the maintained
# runnable 0→hero app; keep it in this CI path so its documented run/chat/inspect
# journey cannot drift from the framework.
go test ./examples/support -run 'TestRunSupportMockSmoke|TestZeroToHeroReadmeDocumentsLifecycle' -count=1
go test ./internal/harness/universe ./internal/harness/plan-delegate -run 'Test.*Harness|TestPlanDelegateEndToEnd|TestPlanDelegateFlowHandoff' -count=1
