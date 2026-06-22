---
layout: default
---

# Migrating from v5 to v6

v6 is a small, mechanical upgrade. The bulk of it is the Go module path; the
behavioral changes are two, both with a one-line fix.

## 1. Module path: `go-micro.dev/v6`

Go puts the major version in the import path, so every import changes:

```go
// before
import "go-micro.dev/v5"
import "go-micro.dev/v5/server"

// after
import "go-micro.dev/v6"
import "go-micro.dev/v6/server"
```

A repo-wide find/replace does it:

```bash
grep -rl 'go-micro.dev/v5' --include='*.go' . \
  | xargs sed -i 's|go-micro.dev/v5|go-micro.dev/v6|g'
go mod tidy
```

Update the CLI too:

```bash
go install go-micro.dev/v6/cmd/micro@v6
```

## 2. TLS is verified by default

In v5, TLS certificate verification was **off** by default (you opted in with
`MICRO_TLS_SECURE=true`). In v6 it is **on** by default — the safe choice now
that an agent, not just a human on a trusted network, can reach an endpoint.

- **Production:** nothing to do. Verification is on.
- **`MICRO_TLS_SECURE` is gone** — remove it; it's the default now.
- **Self-signed certs (local/dev):** opt out with `MICRO_TLS_INSECURE=true`, or
  call `tls.InsecureConfig()` directly.

## 3. `NewService` is the service constructor

The service constructor is now symmetric with `NewAgent` and `NewFlow`:

```go
service := micro.NewService("greeter", micro.Address(":8080"))
agent   := micro.NewAgent("task-mgr", micro.AgentServices("task"))
flow    := micro.NewFlow("onboard", micro.FlowTrigger("events.user.created"))
```

- `micro.New("greeter", ...)` still works as a **deprecated alias** — no rush,
  but prefer `NewService`.
- The old name-less form `micro.NewService(micro.Name("greeter"), ...)` is
  **removed**; pass the name positionally: `micro.NewService("greeter", ...)`.

Generated services already use `NewService` — re-running `micro new` or
`micro run --prompt` emits the v6 form.

## That's it

No other API changed. Agents, services, flows, the registry/broker/store
interfaces, MCP, A2A, and x402 all work as they did — just under
`go-micro.dev/v6` and secure by default.
