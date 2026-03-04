package template

var (
	Readme = `# {{title .Alias}} Service

Generated with

` + "```" + `
micro new {{.Alias}}
` + "```" + `

## Getting Started

Generate the proto code:

` + "```bash" + `
make proto
` + "```" + `

Run the service:

` + "```bash" + `
go run .
` + "```" + `

## MCP & AI Agents

This service is MCP-enabled by default. When running, AI agents can discover
and call your service endpoints automatically.

**MCP tools endpoint:** http://localhost:3001/mcp/tools

### Test with curl

` + "```bash" + `
# List available tools
curl http://localhost:3001/mcp/tools | jq

# Call the service via MCP
curl -X POST http://localhost:3001/mcp/call \
  -H 'Content-Type: application/json' \
  -d '{"tool": "{{lower .Alias}}.{{title .Alias}}.Call", "arguments": {"name": "Alice"}}'
` + "```" + `

### Use with Claude Code

` + "```bash" + `
# Start MCP server for Claude Code
micro mcp serve
` + "```" + `

Or add to your Claude Code config:

` + "```json" + `
{
  "mcpServers": {
    "{{lower .Alias}}": {
      "command": "micro",
      "args": ["mcp", "serve"]
    }
  }
}
` + "```" + `

### Writing Good Tool Descriptions

AI agents work best when your handler methods have clear doc comments:

` + "```go" + `
// CreateUser registers a new user account with the given email and name.
// Returns the created user with their assigned ID.
//
// @example {"email": "alice@example.com", "name": "Alice Smith"}
func (s *Users) CreateUser(ctx context.Context, req *CreateRequest, rsp *CreateResponse) error {
    // ...
}
` + "```" + `

See the [tool descriptions guide](https://go-micro.dev/docs/guides/tool-descriptions) for more tips.

## Development

` + "```bash" + `
make proto    # Regenerate proto code
make build    # Build binary
make test     # Run tests
make dev      # Run with hot reload (requires air)
` + "```" + `
`
)
