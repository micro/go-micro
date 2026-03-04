---
layout: default
title: MCP Troubleshooting
---

# MCP Troubleshooting

Common issues when using the MCP gateway and AI agents with Go Micro services.

## Agent Can't Find My Tools

**Symptom:** Agent says "no tools available" or doesn't list your service endpoints.

**Check 1: Is the service registered?**

```bash
# List registered services
micro services
```

If your service isn't listed, it hasn't registered with the registry. Make sure your service is running and using the same registry as the MCP gateway.

**Check 2: Is the MCP gateway discovering services?**

```bash
# List tools the gateway sees
curl http://localhost:3001/mcp/tools | jq
```

If empty, the gateway can't reach the registry. Verify both use the same registry address.

**Check 3: Are you using the right port?**

The MCP gateway runs on its own port (default `:3001` with `WithMCP`), separate from the service RPC port. Make sure you're querying the MCP port, not the service port.

## Tool Calls Return Errors

**Symptom:** Agent calls a tool but gets an error response.

**"service not found"**

The MCP gateway found the tool definition but can't reach the service. The service may have stopped since the gateway cached its tools. Restart the service and try again.

**"method not found"**

The handler method name doesn't match what the gateway expects. Ensure your handler is properly registered:

```go
// Correct - registers all methods on the handler
service.Handle(new(MyHandler))

// Or with proto-generated code
pb.RegisterMyServiceHandler(service.Server(), handler.New())
```

**"unauthorized" or "forbidden"**

Auth scopes are configured but the agent's token doesn't have the required scope. Check your scope configuration:

```go
// Gateway-side scopes
mcp.Options{
    Scopes: map[string][]string{
        "myservice.Users.Delete": {"users:admin"},
    },
}
```

Verify the agent's bearer token includes the required scopes.

**"rate limited"**

The agent is making too many requests. Adjust rate limits:

```go
mcp.Options{
    RateLimit: &mcp.RateLimitConfig{
        RequestsPerSecond: 100,  // Increase if needed
        Burst:             200,
    },
}
```

## Agent Makes Bad Tool Calls

**Symptom:** Agent calls tools with wrong parameters or misunderstands what a tool does.

This is almost always a documentation problem. Improve your handler doc comments:

```go
// Bad - agent doesn't know what this does
func (s *Users) Get(ctx context.Context, req *GetRequest, rsp *GetResponse) error {

// Good - agent understands purpose, parameters, and format
// Get retrieves a user by their unique ID. Returns the full user profile
// including email, display name, and account status.
//
// @example {"id": "user-123"}
func (s *Users) Get(ctx context.Context, req *GetRequest, rsp *GetResponse) error {
```

Add `description` struct tags to your request/response types:

```go
type GetRequest struct {
    ID string `json:"id" description:"User ID in UUID format"`
}
```

See the [Tool Descriptions Guide](tool-descriptions.md) for detailed best practices.

## WebSocket Connection Drops

**Symptom:** WebSocket connections to `ws://localhost:3001/mcp/ws` disconnect unexpectedly.

**Check 1:** Make sure your client sends periodic pings. The WebSocket transport expects heartbeats to detect stale connections.

**Check 2:** If running behind a reverse proxy (nginx, Caddy), ensure WebSocket upgrade headers are forwarded:

```nginx
location /mcp/ws {
    proxy_pass http://localhost:3001;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 3600s;
}
```

**Check 3:** Check for connection limits. Each WebSocket connection is persistent. If you have many agents, you may need to increase file descriptor limits.

## Claude Code Can't Connect

**Symptom:** Claude Code doesn't see your MCP tools after configuring the server.

**Check 1: Test stdio transport manually**

```bash
# This should start and wait for JSON-RPC input
micro mcp serve
```

If it errors, check that your services are running and the registry is accessible.

**Check 2: Verify config syntax**

In your Claude Code MCP settings:

```json
{
  "mcpServers": {
    "my-services": {
      "command": "micro",
      "args": ["mcp", "serve"]
    }
  }
}
```

Common mistakes:
- Wrong path to `micro` binary (use absolute path if needed)
- Missing `"serve"` in args
- Service not running when Claude Code starts

**Check 3: Check micro is in PATH**

```bash
which micro
```

If not found, use the full path in your config:

```json
{
  "mcpServers": {
    "my-services": {
      "command": "/usr/local/bin/micro",
      "args": ["mcp", "serve"]
    }
  }
}
```

## OpenTelemetry Traces Missing

**Symptom:** MCP gateway calls aren't showing up in your trace collector.

The gateway only creates real spans when a `TraceProvider` is configured:

```go
mcp.Options{
    TraceProvider: otel.GetTracerProvider(),
}
```

Without this, noop spans are used (no traces exported). Make sure you've initialized the OpenTelemetry SDK before starting the gateway.

## Audit Logs Not Appearing

**Symptom:** No audit records despite tool calls succeeding.

Audit logging requires an explicit callback:

```go
mcp.Options{
    AuditFunc: func(r mcp.AuditRecord) {
        log.Printf("[audit] tool=%s account=%s allowed=%t duration=%s",
            r.Tool, r.AccountID, r.Allowed, r.Duration)
    },
}
```

If `AuditFunc` is nil, no audit records are generated.

## Performance Issues

**Symptom:** MCP tool calls are slow.

**Check 1: Network round-trips**

Each MCP tool call makes an RPC call to the underlying service. If the service is on a different host, network latency applies. Use `micro mcp test` to measure raw latency.

**Check 2: Service discovery caching**

The gateway caches service/tool metadata. If you're seeing stale data, it's because of caching. The cache refreshes periodically based on registry TTL.

**Check 3: Rate limiting**

If rate limits are too low, requests queue up. Check your rate limit configuration.

## Still Stuck?

- Check the [MCP Documentation](../../mcp.md) for full API reference
- Search [GitHub Issues](https://github.com/micro/go-micro/issues) for similar problems
- Ask in [GitHub Discussions](https://github.com/micro/go-micro/discussions)
