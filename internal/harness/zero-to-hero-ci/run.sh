#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

run_step() {
  local name=$1
  shift

  printf '\n==> %s\n' "$name"
  printf '+ %q' "$@"
  printf '\n'
  "$@"
}

# Keep the developer inner-loop boundaries executable and discoverable in CI
# without secrets or long-running daemons. Step names mirror the documented
# install → scaffold → run/chat → inspect → deploy-dry-run seams so failures
# identify the broken part of the getting-started contract.
run_step "scaffold: 0→1 service contract" \
  go test ./cmd/micro/cli/new -run TestZeroToOne -count=1
run_step "run/chat/inspect: first-agent CLI boundaries" \
  go test ./cmd/micro -run 'TestFirstAgentWalkthroughCLIBoundaries|TestExamplesWayfindingIndexStaysLinked|TestExamplesCommandPointsAtWayfindingIndex|TestZeroToHeroCLIBoundaries|TestZeroToHeroCommandPrintsMaintainedNoSecretPath' -count=1
run_step "deploy dry-run: configured target plan" \
  go test ./cmd/micro/cli/deploy -run TestDeployDryRun -count=1
run_step "chat/inspect: no-secret first-agent transcript and docs" \
  go test ./internal/harness/zero-to-hero-ci -run 'TestNoSecretFirstAgentTranscript|TestNoSecretFirstAgentDebuggingSmoke|TestZeroToHeroReferenceDocs|TestZeroToHeroDeployDryRunCommandSmoke|TestYourFirstAgentTutorialSmoke' -count=1

# Deterministic no-secret reference scenarios. These use the real Go Micro
# runtime and mock only the LLM provider. The support example is the maintained
# runnable 0→hero app; keep it in this CI path so its documented run/chat/inspect
# journey cannot drift from the framework.
run_step "first-agent app: runnable provider-free example" \
  go test ./examples/first-agent -run TestRunFirstAgent -count=1
run_step "0→hero app: support lifecycle smoke" \
  go test ./examples/support -run 'TestRunSupportMockSmoke|TestZeroToHeroReadmeDocumentsLifecycle' -count=1
run_step "flow history: deterministic services → agents → workflows harnesses" \
  go test ./internal/harness/universe ./internal/harness/plan-delegate -run 'Test.*Harness|TestPlanDelegateEndToEnd|TestPlanDelegateFlowHandoff' -count=1
