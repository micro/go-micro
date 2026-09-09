# MCP CLI Command Examples

This document provides examples of using the `micro mcp` commands for AI agent integration.

## Table of Contents

- [List Available Tools](#list-available-tools)
- [Test a Tool](#test-a-tool)
- [Generate Documentation](#generate-documentation)
- [Export to Different Formats](#export-to-different-formats)

## Prerequisites

You need at least one microservice running with the go-micro framework. The service will automatically be discovered via the registry (mdns by default).

Example service:
```bash
cd examples/mcp/hello
go run main.go
```

## List Available Tools

### Human-readable list
```bash
micro mcp list
```

Output:
```
Available MCP Tools:

Service: greeter
  • greeter.Greeter.SayHello

Total: 1 tools
```

### JSON output
```bash
micro mcp list --json
```

Output:
```json
{
  "count": 1,
  "tools": [
    {
      "description": "Call SayHello on greeter service",
      "endpoint": "Greeter.SayHello",
      "name": "greeter.Greeter.SayHello",
      "service": "greeter"
    }
  ]
}
```

## Test a Tool

### Basic test
```bash
micro mcp test greeter.Greeter.SayHello '{"name": "Alice"}'
```

Output:
```
Testing tool: greeter.Greeter.SayHello
Service: greeter
Endpoint: Greeter.SayHello
Input: {"name": "Alice"}

✅ Call successful!

Response:
{
  "message": "Hello Alice!"
}
```

### Test with default empty input
```bash
micro mcp test greeter.Greeter.SayHello
```

This will call the tool with an empty JSON object `{}`.

## Generate Documentation

### Markdown documentation (stdout)
```bash
micro mcp docs
```

Output:
```markdown
# MCP Tools Documentation

Generated: 2026-02-13 14:30:00

Total Tools: 1

## Service: greeter

### greeter.Greeter.SayHello

**Description:** Greets a person by name. Returns a friendly greeting message.

**Example Input:**
\`\`\`json
{"name": "Alice"}
\`\`\`
```

### Markdown documentation (save to file)
```bash
micro mcp docs --output mcp-tools.md
```

This creates a `mcp-tools.md` file with the documentation.

### JSON documentation
```bash
micro mcp docs --format json
```

Output:
```json
{
  "count": 1,
  "tools": [
    {
      "description": "Greets a person by name. Returns a friendly greeting message.",
      "endpoint": "Greeter.SayHello",
      "example": "{\"name\": \"Alice\"}",
      "metadata": {
        "description": "Greets a person by name. Returns a friendly greeting message.",
        "example": "{\"name\": \"Alice\"}"
      },
      "name": "greeter.Greeter.SayHello",
      "scopes": null,
      "service": "greeter"
    }
  ]
}
```

### JSON documentation (save to file)
```bash
micro mcp docs --format json --output tools.json
```

## Export to Different Formats

### Export to LangChain (Python)

Generate Python code with LangChain tool definitions:

```bash
micro mcp export langchain
```

Output:
```python
# LangChain Tools for Go Micro Services
# Auto-generated from MCP service discovery

from langchain.tools import Tool
import requests
import json

# Configure your MCP gateway endpoint
MCP_GATEWAY_URL = 'http://localhost:3000/mcp'

def call_mcp_tool(tool_name, arguments):
    """Call an MCP tool via HTTP gateway"""
    response = requests.post(
        f'{MCP_GATEWAY_URL}/call',
        json={'name': tool_name, 'arguments': arguments}
    )
    response.raise_for_status()
    return response.json()

# Define tools
tools = []

def greeter_Greeter_SayHello(arguments: str) -> str:
    """Greets a person by name. Returns a friendly greeting message."""
    args = json.loads(arguments) if isinstance(arguments, str) else arguments
    return json.dumps(call_mcp_tool('greeter.Greeter.SayHello', args))

tools.append(Tool(
    name='greeter.Greeter.SayHello',
    func=greeter_Greeter_SayHello,
    description='Greets a person by name. Returns a friendly greeting message.'
))

# Example usage:
# from langchain.agents import initialize_agent, AgentType
# from langchain.llms import OpenAI
#
# llm = OpenAI(temperature=0)
# agent = initialize_agent(tools, llm, agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION)
# agent.run('Your query here')
```

Save to file:
```bash
micro mcp export langchain --output langchain_tools.py
```

### Export to OpenAPI 3.0

Generate an OpenAPI specification:

```bash
micro mcp export openapi
```

Output:
```json
{
  "components": {
    "securitySchemes": {
      "bearerAuth": {
        "scheme": "bearer",
        "type": "http"
      }
    }
  },
  "info": {
    "description": "Auto-generated OpenAPI spec from MCP service discovery",
    "title": "Go Micro MCP Services",
    "version": "1.0.0"
  },
  "openapi": "3.0.0",
  "paths": {
    "/mcp/call/greeter/Greeter/SayHello": {
      "post": {
        "description": "Greets a person by name. Returns a friendly greeting message.",
        "operationId": "greeter_Greeter_SayHello",
        "requestBody": {
          "content": {
            "application/json": {
              "schema": {
                "type": "object"
              }
            }
          },
          "required": true
        },
        "responses": {
          "200": {
            "content": {
              "application/json": {
                "schema": {
                  "type": "object"
                }
              }
            },
            "description": "Successful response"
          }
        },
        "summary": "greeter.Greeter.SayHello"
      }
    }
  },
  "servers": [
    {
      "description": "MCP Gateway",
      "url": "http://localhost:3000"
    }
  ]
}
```

Save to file:
```bash
micro mcp export openapi --output openapi.json
```

### Export to raw JSON

Export raw tool definitions:

```bash
micro mcp export json
```

This is similar to `micro mcp docs --format json` but specifically for export purposes.

Save to file:
```bash
micro mcp export json --output tools.json
```

## Using with Different Registries

By default, the commands use mdns registry. You can specify a different registry:

```bash
# Using consul
micro mcp list --registry consul --registry_address consul:8500

# Using etcd
micro mcp list --registry etcd --registry_address etcd:2379
```

## Integration Examples

### Using LangChain Export with Claude

1. Export your tools to LangChain format:
```bash
micro mcp export langchain --output my_tools.py
```

2. Use in your Python agent:
```python
from my_tools import tools
from langchain.agents import initialize_agent, AgentType
from langchain.chat_models import ChatAnthropic

llm = ChatAnthropic(model="claude-3-sonnet-20240229")
agent = initialize_agent(tools, llm, agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION)

result = agent.run("Greet Alice")
print(result)
```

### Using OpenAPI Export with GPT

1. Export to OpenAPI:
```bash
micro mcp export openapi --output openapi.json
```

2. Upload to ChatGPT as a custom GPT action or use with OpenAI Assistants API.

### Documentation for AI Agents

Generate documentation that AI agents can read to understand your services:

```bash
micro mcp docs --format json --output service-catalog.json
```

This JSON file can be fed to AI agents for service discovery and understanding.

## Advanced Usage

### Piping and Processing

You can pipe the output to other tools:

```bash
# Count tools per service
micro mcp list --json | jq '.tools | group_by(.service) | map({service: .[0].service, count: length})'

# Extract all tool names
micro mcp list --json | jq -r '.tools[].name'

# Filter tools by service
micro mcp list --json | jq '.tools[] | select(.service == "greeter")'
```

### Monitoring and CI/CD

Use these commands in your CI/CD pipeline:

```bash
# Validate all services are discoverable
SERVICE_COUNT=$(micro mcp list --json | jq '.count')
if [ "$SERVICE_COUNT" -lt 5 ]; then
  echo "Error: Expected at least 5 services, found $SERVICE_COUNT"
  exit 1
fi

# Generate documentation on each deployment
micro mcp docs --output docs/mcp-services.md
git add docs/mcp-services.md
git commit -m "Update MCP service documentation"
```

### Testing in Development

Create a script to test all your tools:

```bash
#!/bin/bash
# test-all-tools.sh

TOOLS=$(micro mcp list --json | jq -r '.tools[].name')

for tool in $TOOLS; do
  echo "Testing $tool..."
  micro mcp test "$tool" "{}" || echo "Failed: $tool"
done
```

## Troubleshooting

### No tools found

If `micro mcp list` shows 0 tools:

1. Verify services are running:
```bash
ps aux | grep "your-service"
```

2. Check registry (mdns might need time to discover):
```bash
# Wait a few seconds and try again
sleep 3
micro mcp list
```

3. Use a different registry if mdns is unreliable:
```bash
# Start services with consul
micro --registry consul server

# List with consul
micro mcp list --registry consul
```

### Service not responding in tests

If `micro mcp test` fails:

1. Verify the tool name is correct:
```bash
micro mcp list
```

2. Check the JSON input format:
```bash
# Invalid
micro mcp test service.Handler.Method '{invalid}'

# Valid
micro mcp test service.Handler.Method '{"key": "value"}'
```

3. Check service logs for errors.

## Next Steps

- Read the [MCP Documentation](../../gateway/mcp/DOCUMENTATION.md)
- Try the [MCP Examples](../../examples/mcp/README.md)
- Learn about [Tool Scopes and Security](../../gateway/mcp/DOCUMENTATION.md#authentication-and-scopes)
- Explore [Agent SDKs](#) (coming soon)
