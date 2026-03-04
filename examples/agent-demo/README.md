# Agent Demo

A multi-service project management app that demonstrates AI agents interacting with Go Micro services through MCP.

## What's Included

Three services registered in a single process:

| Service | Endpoints | Description |
|---------|-----------|-------------|
| **ProjectService** | Create, Get, List | Manage projects with status tracking |
| **TaskService** | Create, List, Update | Tasks with assignees, priorities, and status |
| **TeamService** | Add, List, Get | Team members with roles and skills |

The demo starts with seed data: 2 projects, 7 tasks, and 4 team members.

## Run

```bash
go run main.go
```

Endpoints:
- **MCP Gateway:** http://localhost:3000
- **MCP Tools:** http://localhost:3000/mcp/tools
- **WebSocket:** ws://localhost:3000/mcp/ws

## Use with Claude Code

```json
{
  "mcpServers": {
    "demo": {
      "command": "go",
      "args": ["run", "main.go"],
      "cwd": "examples/agent-demo"
    }
  }
}
```

## Example Prompts

Try these with Claude Code or any MCP client:

- "What projects do we have?"
- "Show me all tasks assigned to alice"
- "Create a high-priority task for bob to review the design mockups"
- "Who on the team knows Go?"
- "Give me a status update on the Website Redesign project"
- "What tasks are still todo on the API v2 migration?"
- "Assign the unassigned tasks to charlie"
- "Mark task-1 as done"

## What This Demonstrates

1. **Zero-config MCP** — Services become AI tools automatically from doc comments
2. **Cross-service orchestration** — An agent queries projects, tasks, and team in one conversation
3. **Rich tool descriptions** — `description` struct tags and `@example` comments guide the agent
4. **Auth scopes** — Read and write operations have separate scopes
5. **`WithMCP` one-liner** — MCP gateway starts with a single option

See the [blog post](/blog/4) for a detailed walkthrough.
