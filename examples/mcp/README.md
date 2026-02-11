# MCP Examples

Examples demonstrating Model Context Protocol (MCP) integration with go-micro.

## Examples

### [hello](./hello/) - Minimal Example ⭐ Start Here

The simplest possible MCP-enabled service. Perfect for learning the basics.

**What it shows:**
- Automatic documentation extraction from Go comments
- MCP gateway setup with 3 lines
- Ready for Claude Code

**Run it:**
```bash
cd hello
go run main.go
```

### [documented](./documented/) - Full-Featured Example

Complete example showing all MCP features with a user service.

**What it shows:**
- Multiple endpoints (GetUser, CreateUser)
- Rich documentation with examples
- Pre-populated test data
- Production-ready patterns

**Run it:**
```bash
cd documented
go run main.go
```

## Quick Start

### 1. Write Your Service

Add Go doc comments to your handler methods:

```go
// SayHello greets a person by name. Returns a friendly greeting message.
//
// @example {"name": "Alice"}
func (g *Greeter) SayHello(ctx context.Context, req *HelloRequest, rsp *HelloResponse) error {
    rsp.Message = "Hello " + req.Name + "!"
    return nil
}

type HelloRequest struct {
    Name string `json:"name" description:"Person's name to greet"`
}
```

### 2. Register Handler (Auto-Extracts Docs!)

```go
handler := service.Server().NewHandler(new(Greeter))
service.Server().Handle(handler)
```

### 3. Start MCP Gateway

```go
go mcp.ListenAndServe(":3000", mcp.Options{
    Registry: service.Options().Registry,
})
```

## Testing

### HTTP API

```bash
# List tools
curl http://localhost:3000/mcp/tools | jq

# Call a tool
curl -X POST http://localhost:3000/mcp/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "greeter.Greeter.SayHello",
    "input": {"name": "Alice"}
  }' | jq
```

### Claude Code (Stdio)

Start MCP server:
```bash
micro mcp serve
```

Add to `~/.claude/claude_desktop_config.json`:
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

Restart Claude Code and ask Claude to use your services!

## Features

### ✅ Automatic Documentation Extraction

Just write Go comments - documentation is extracted automatically:

- **Go doc comments** → Tool descriptions
- **@example tags** → Example inputs for AI
- **Struct tags** → Parameter descriptions

### ✅ Multiple Transports

- **Stdio** - For Claude Code (recommended)
- **HTTP/SSE** - For web-based agents

### ✅ MCP Command Line

```bash
micro mcp serve              # Start with stdio
micro mcp serve --address :3000  # Start with HTTP
micro mcp list               # List tools
micro mcp test <tool-name>   # Test a tool
```

### ✅ Zero Configuration

- No manual tool registration
- No API wrappers
- No code generation
- Just write normal Go code!

## Documentation

- [Full MCP Documentation](../../internal/website/docs/mcp.md)
- [MCP Gateway Implementation](../../gateway/mcp/)
- [Documentation Guide](../../gateway/mcp/DOCUMENTATION.md)
- [Blog Post](../../internal/website/blog/2.md)

## Learn More

- [Model Context Protocol Spec](https://modelcontextprotocol.io/)
- [Go Micro Documentation](https://go-micro.dev)
