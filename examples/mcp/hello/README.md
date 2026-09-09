# MCP Hello World Example

The simplest possible MCP-enabled go-micro service.

## What This Shows

- ✅ Automatic documentation extraction from Go comments
- ✅ MCP gateway setup with 3 lines of code
- ✅ Ready for Claude Code integration
- ✅ HTTP endpoint for testing

## Run It

```bash
cd examples/mcp/hello
go run main.go
```

## Test It

### Option 1: HTTP API

```bash
# List available tools
curl http://localhost:3000/mcp/tools | jq

# Call the SayHello tool
curl -X POST http://localhost:3000/mcp/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "greeter.Greeter.SayHello",
    "input": {"name": "Alice"}
  }' | jq
```

### Option 2: Claude Code

In a separate terminal:

```bash
micro mcp serve
```

Add to `~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "greeter": {
      "command": "micro",
      "args": ["mcp", "serve"]
    }
  }
}
```

Restart Claude Code and ask:

> "Say hello to Bob using the greeter service"

## How It Works

### 1. Write Normal Go Code

```go
// SayHello greets a person by name. Returns a friendly greeting message.
//
// @example {"name": "Alice"}
func (g *Greeter) SayHello(ctx context.Context, req *HelloRequest, rsp *HelloResponse) error {
    rsp.Message = "Hello " + req.Name + "!"
    return nil
}
```

### 2. Register the Handler

```go
// Documentation is extracted automatically!
handler := service.Server().NewHandler(new(Greeter))
service.Server().Handle(handler)
```

### 3. Start MCP Gateway

```go
go mcp.ListenAndServe(":3000", mcp.Options{
    Registry: service.Options().Registry,
})
```

**That's it!** Your service is now AI-accessible.

## What Gets Extracted

From this code:

```go
// SayHello greets a person by name. Returns a friendly greeting message.
//
// @example {"name": "Alice"}
func (g *Greeter) SayHello(...)

type HelloRequest struct {
    Name string `json:"name" description:"Person's name to greet"`
}
```

Claude sees:

```json
{
  "name": "greeter.Greeter.SayHello",
  "description": "SayHello greets a person by name. Returns a friendly greeting message.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": {
        "type": "string",
        "description": "Person's name to greet"
      }
    },
    "examples": ["{\"name\": \"Alice\"}"]
  }
}
```

## Next Steps

- See `examples/mcp/documented` for a more complete example with multiple endpoints
- Read `/docs/mcp.md` for full documentation
- Check out the [MCP specification](https://modelcontextprotocol.io/)
