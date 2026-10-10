#!/usr/bin/env bash
# Real terminal + loopback model. No provider credentials or external services.
set -euo pipefail
cd "$(dirname "$0")/.."
chat_tmp=$(mktemp -d)
trap 'rm -rf "$chat_tmp"' EXIT
go build -o "$chat_tmp/micro" ./cmd/micro
python3 internal/harness/chat/run.py "$chat_tmp/micro"
