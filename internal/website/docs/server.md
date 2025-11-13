---
layout: default
---

# Micro Server (Optional)

The Micro server is an optional API and dashboard that provides a fixed entrypoint for discovering and interacting with services. It is not required to build or run services; the examples in this documentation run services directly with `go run`.

## Install

Install the CLI which includes the server command:

```
go install go-micro.dev/v5/cmd/micro@latest
```

## Run

Start the server:

```
micro server
```

Then open http://localhost:8080 in your browser.

## When to use it
- Exploring registered services and endpoints
- Calling endpoints via a web UI or HTTP API
- Local development and debugging

Note: The server is evolving and configuration or features may change. For CLI usage details, see `cmd/micro/cli/README.md` in this repository.
