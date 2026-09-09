#!/usr/bin/env bash
# Smoke-test the documented install.sh path without network access.
# It builds the local CLI, packages it like a release archive, installs it into a
# temporary bin directory through internal/scripts/install.sh, then checks the
# first-run command boundaries shown in the getting-started docs.

set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

ARCHIVE_DIR="$TMP_DIR/archive"
INSTALL_DIR="$TMP_DIR/install"
ARCHIVE="$TMP_DIR/micro-local.tar.gz"
mkdir -p "$ARCHIVE_DIR" "$INSTALL_DIR"

CGO_ENABLED=0 go build -o "$ARCHIVE_DIR/micro" ./cmd/micro
chmod +x "$ARCHIVE_DIR/micro"
tar -C "$ARCHIVE_DIR" -czf "$ARCHIVE" micro

MICRO_INSTALL_DIR="$INSTALL_DIR" \
MICRO_INSTALL_ARCHIVE="$ARCHIVE" \
MICRO_VERSION="local-smoke" \
PATH="$INSTALL_DIR:$PATH" \
  "$ROOT/internal/scripts/install.sh" > "$TMP_DIR/install.out"

MICRO="$INSTALL_DIR/micro"
if [[ ! -x "$MICRO" ]]; then
  echo "installed micro binary not found at $MICRO" >&2
  cat "$TMP_DIR/install.out" >&2
  exit 1
fi

require_output() {
  local description=$1
  local expected=$2
  shift 2
  local output
  if ! output=$("$MICRO" "$@" 2>&1); then
    echo "micro $* failed while checking $description" >&2
    echo "$output" >&2
    exit 1
  fi
  if [[ "$output" != *"$expected"* ]]; then
    echo "micro $* missing expected text '$expected' for $description" >&2
    echo "$output" >&2
    exit 1
  fi
}

require_ordered_output() {
  local description=$1
  shift
  local -a expected=()
  while [[ $# -gt 0 && "$1" != "--" ]]; do
    expected+=("$1")
    shift
  done
  shift

  local output
  if ! output=$("$MICRO" "$@" 2>&1); then
    echo "micro $* failed while checking $description" >&2
    echo "$output" >&2
    exit 1
  fi

  local remainder=$output
  for text in "${expected[@]}"; do
    if [[ "$remainder" != *"$text"* ]]; then
      echo "micro $* missing expected ordered text '$text' for $description" >&2
      echo "$output" >&2
      exit 1
    fi
    remainder=${remainder#*"$text"}
  done
}

require_output "version" "micro version" --version
require_output "root help" "COMMANDS" --help
require_output "service scaffold" "micro new" new --help
require_output "first-agent preflight" "preflight" agent preflight --help
require_output "local runtime" "micro run" run --help
require_output "agent chat" "micro chat" chat --help
require_output "agent inspection" "micro inspect agent" inspect agent --help
require_output "flow inspection" "micro inspect flow" inspect flow --help

require_ordered_output "installed first-agent docs wayfinding" \
  "micro agent demo" \
  "no-secret-first-agent.html" \
  "your-first-agent.html" \
  "micro agent preflight  # before micro run: prerequisites" \
  "micro run" \
  "micro chat" \
  "micro agent doctor     # after micro run: chat/gateway/inspect recovery" \
  "debugging-agents.html" \
  "micro inspect agent <name>" \
  "zero-to-hero.html" \
  -- docs

require_ordered_output "installed provider-free examples wayfinding" \
  "go run ./examples/first-agent" \
  "go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentTranscript -count=1" \
  "go run ./examples/support" \
  "micro agent demo" \
  "micro docs" \
  "micro zero-to-hero" \
  "no-secret-first-agent.html" \
  "your-first-agent.html" \
  "debugging-agents.html" \
  "zero-to-hero.html" \
  -- examples

require_ordered_output "installed no-secret agent demo" \
  "provider-free" \
  "go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentTranscript -count=1" \
  "your-first-agent.html" \
  "debugging-agents.html" \
  "zero-to-hero.html" \
  "micro agent preflight  # before micro run: prerequisites" \
  "micro run" \
  "micro chat" \
  "micro agent doctor     # after micro run: chat/gateway/inspect recovery" \
  "micro inspect agent <name>" \
  -- agent demo

require_ordered_output "installed zero-to-hero lifecycle wayfinding" \
  "./internal/harness/zero-to-hero-ci/run.sh" \
  "go run ./examples/first-agent" \
  "go run ./examples/support" \
  "make harness" \
  "zero-to-hero.html" \
  -- zero-to-hero

echo "✓ install smoke path verified"
