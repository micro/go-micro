---
title: Documentation
linkTitle: Docs
params:
  docs_main: true
  exclude_search: false
cascade:
  params:
    exclude_search: true
menu:
  main:
    weight: 10
---

Go Micro is a Go framework for services and agents.

Choose a starting point:

- [Build and run a service](quickstart.md) — Go code and RPC; no model needed.
- [Expose services to agents](guides/services-with-claude-code.md) — connect Claude Code through MCP.
- [Chat with services](guides/chat-with-services.md) — use Go Micro’s terminal agent, without a gateway.

For extending your application:

- [Services](services.md): expose and call application operations.
- [Agents](agents.md): use services as tools through a model.

[Reference](reference.md) covers the individual packages and CLI.
