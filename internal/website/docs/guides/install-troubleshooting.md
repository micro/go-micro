---
layout: default
title: Install troubleshooting
---

# Install troubleshooting

Use this page before `micro new` or `micro agent demo` when the CLI install is
unclear. The goal is to prove three boundaries in order: the `micro` binary is on
`PATH`, it is the version you expected, and the no-secret first-run path works
without provider keys.

## 1. Choose one install path

### Binary installer (no Go required to install)

```sh
curl -fsSL https://go-micro.dev/install.sh | sh
```

Use this when you want the released `micro` binary without building it yourself.
The generated services still need a Go toolchain when you run `micro run`, but the
installer itself does not require Go.

### Go install (build from source)

```sh
go install go-micro.dev/v6/cmd/micro@latest
```

Use this when Go is already installed and you want the binary in your Go bin
directory. If the command succeeds but `micro` is not found, your Go bin directory
is probably not on `PATH`.

## 2. Verify `PATH` and version

Check which binary your shell will run:

```sh
command -v micro
micro --version
```

If `command -v micro` prints nothing, add the install directory to `PATH`, then
open a new terminal and retry. Common locations are:

```sh
export PATH="$HOME/.micro/bin:$PATH"      # binary installer
export PATH="$(go env GOPATH)/bin:$PATH"  # go install
```

If `micro --version` shows an older binary than expected, remove the stale copy or
put the intended install directory earlier in `PATH`.

## 3. Run the no-secret smoke path

Once `micro` resolves, prove the local service runtime before adding LLM provider
keys:

```sh
micro new helloworld
cd helloworld
micro run
```

In another terminal:

```sh
curl -X POST http://localhost:8080/api/helloworld/Helloworld.Call \
  -H 'Content-Type: application/json' -d '{"name":"World"}'
```

This checks the scaffold, local build, gateway, and service registration without
calling a model provider.

## 4. Recover common failures

| Symptom | Check | Fix |
|---------|-------|-----|
| `micro: command not found` | `command -v micro` | Add the installer bin directory or `$(go env GOPATH)/bin` to `PATH`, then open a new terminal. |
| `micro run` cannot find Go | `go version` | Install Go 1.24 or newer from <https://go.dev/doc/install>. |
| The gateway port is busy | `lsof -i :8080` | Stop the process using the port, or run with a different address. |
| Provider-key errors block an agent run | `micro agent preflight` | Stay on the no-secret path first: run `micro agent demo`, then the no-secret first-agent guide. |

## 5. Continue the first-agent on-ramp

After install verification succeeds, continue in order:

1. `micro agent demo` — print the provider-free first-agent demo command and next docs steps.
2. [No-secret first-agent transcript](no-secret-first-agent.html) — prove an agent can use services without a provider key.
3. [Your First Agent](your-first-agent.html) — build and chat with your own service-backed agent.
4. [Debugging your agent](debugging-agents.html) — inspect registration, tool calls, run history, and provider failures.
5. [0→hero Reference](zero-to-hero.html) — walk the full services → agents → workflows lifecycle.

For repository contributors, `make install-smoke` runs the same installer seam
against a local build without network access.
