---
title: "Expose services to Claude Code"
params:
  exclude_search: false
---

Connect Claude Code to a running Go Micro service through MCP. You do not need
`micro chat` or a Go Micro agent. Claude Code supplies the model and agent;
the gateway makes service operations available as tools.

## Before you start

Use an installed, signed-in [Claude Code](https://code.claude.com/docs/en/overview).
Install the Go Micro CLI if you do not already have v6.15.1 or newer:

```sh
go install go-micro.dev/v6/cmd/micro@v6.15.1
export PATH="$(go env GOPATH)/bin:$PATH"
```

Keep your service running. The commands below use the `notes` service from the
[service quickstart](../quickstart.md); substitute your service and endpoint names.

First check that the CLI can reach it:

```sh
micro call notes Notes.List '{}'
```

If discovery fails, check that the CLI and service use the same registry.
See [service discovery](../services.md#discovery) for NATS configuration.

## Start the gateway

In another terminal:

```sh
micro gateway --address 127.0.0.1:8080 \
  --mcp-address 127.0.0.1:3000 \
  --mcp-tool notes.Notes.List
```

Leave it running. The service and gateway are separate processes.
No model key is needed by the gateway.

## Connect Claude Code

In your Claude Code project directory, add the local HTTP endpoint:

```sh
claude mcp add --transport http go-micro http://127.0.0.1:3000/mcp
claude mcp list
claude
```

Inside Claude Code, use `/mcp` to check the connection and available tools.
Ask **“Use the go-micro MCP server to list my notes.”** Approve the tool call
if prompted. Inspect the tool result: the quickstart service returns
“Review the release” and “Update the documentation”. This should match the RPC
response above, rather than an answer inferred from reading your source files.

If the connection fails, check that the gateway is running on this machine.
If the tool is missing, check the endpoint spelling and service discovery.
For client configuration details, see [Claude Code’s MCP documentation](https://code.claude.com/docs/en/mcp).

To remove the connection:

```sh
claude mcp remove go-micro
```

## Other clients and remote access

Other Streamable HTTP MCP clients can use the same URL. The loopback address
requires the client to run on this machine; it is not a URL a hosted chat
application can reach.

Loopback listeners above are for local use. When you bind a gateway listener
to a non-loopback address, authentication is enabled by default. Set
`MICRO_AUTH_TOKEN` to supply a bearer token, or use the generated token printed
at startup. Configure the same token in your MCP client's Authorization header.
Use HTTPS for remote access.

`--mcp-tool` selects the MCP operations to expose; repeat it for additional tools.
It does not restrict the HTTP API. Without it, MCP exposes all discovered tools,
including framework operations. See the [gateway reference](https://github.com/micro/go-micro/blob/master/cmd/micro/README.md#micro-gateway)
for authentication and scopes.

The gateway also exposes HTTP independently of MCP:

```sh
curl -s http://127.0.0.1:8080/api/notes/Notes/List \
  -H 'Content-Type: application/json' -d '{}'
```
