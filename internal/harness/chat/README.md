# Chat developer journey

Run `make chat-smoke` on Linux or macOS with Go and Python 3. It builds the CLI,
starts a loopback OpenAI-compatible endpoint, and drives `micro chat` through a
real pseudo-terminal. It uses temporary projects and profiles and does not read
provider credentials from your environment.

The model responses are scripted. Everything below is real:

- First launch, invalid provider choices, hidden credential entry, and model selection.
- Loading `AGENTS.md`, reading a file, approving a write and shell command, and
  checking their filesystem results. Denying a write leaves no file.
- Restarting the process with saved settings and conversation context.
- Receiving an HTTP 401, replacing the key with `/login`, and retrying successfully.
- Interrupting a stream, queued and multiline input, and creating an isolated session.
- Missing credentials in noninteractive mode, provider environment selection,
  and reaching setup with default mDNS discovery.

CI runs `make install-smoke`, which packages the CLI, installs it through the
documented `sh` installer, and runs this journey against the installed binary.
`make chat-smoke` is the shorter development loop. The service-backed example and remote
approval tests cover framework extensions and RPC separately:

```bash
go test -race ./cmd/micro/chat ./agent/workspace ./examples/first-agent
```

This proves CLI plumbing and real tool effects, not live model quality, provider
account access, mDNS discovery between machines, or Docker execution. Before a
release, also run the [live recording](../../demo/README.md) with a supported
provider/model, inspect the actual file change, restart chat, and verify it can
continue the task. Keep that verification separate from the deterministic test.
