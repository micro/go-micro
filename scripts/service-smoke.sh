#!/usr/bin/env bash
# Verify the documented service, gateway and agent with real local processes.
set -euo pipefail
cd "$(dirname "$0")/.."
service_tmp=$(mktemp -d)
trap 'rm -rf "$service_tmp"' EXIT
nats_bin=$(command -v nats-server) || {
  echo 'Install the test dependency: go install github.com/nats-io/nats-server/v2@v2.11.3' >&2
  exit 1
}
version=${1:-}
if [[ -n "$version" ]]; then
  GOBIN="$service_tmp" go install "go-micro.dev/v6/cmd/micro@$version"
else
  go build -o "$service_tmp/micro" ./cmd/micro
fi
python3 internal/harness/services.py "$service_tmp/micro" "$nats_bin" "$version"
