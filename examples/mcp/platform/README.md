# Platform Example: AI Agents Meet Real Microservices

This example mirrors the [micro/blog](https://github.com/micro/blog) platform — a real microblogging application built on Go Micro. It demonstrates how existing microservices become AI-accessible through MCP with **zero changes to business logic**.

## Services

| Service | Endpoints | Description |
|---------|-----------|-------------|
| **Users** | Signup, Login, GetProfile, UpdateStatus, List | Account management and authentication |
| **Posts** | Create, Read, Update, Delete, List, TagPost, UntagPost, ListTags | Blog posts with markdown and tagging |
| **Comments** | Create, List, Delete | Threaded comments on posts |
| **Mail** | Send, Read | Internal messaging between users |

## Running

```bash
go run .
```

MCP tools available at: http://localhost:3001/mcp/tools

## Agent Scenarios

These are realistic multi-step workflows an AI agent can complete:

### 1. New User Onboarding
```
"Sign up a new user called carol, then write a welcome post introducing herself"
```
The agent will: call Signup → use the returned user ID → call Posts.Create

### 2. Content Creation
```
"Log in as alice and write a blog post about Go concurrency patterns, then tag it with 'golang' and 'concurrency'"
```
The agent will: call Login → call Posts.Create → call TagPost twice

### 3. Social Interaction
```
"List all posts, find the welcome post, and comment on it as bob saying 'Great to be here!'"
```
The agent will: call Posts.List → pick the right post → call Comments.Create

### 4. Cross-Service Workflow
```
"Send a mail from alice to bob welcoming him, then check bob's inbox to confirm delivery"
```
The agent will: call Mail.Send → call Mail.Read to verify

### 5. Platform Overview
```
"Show me all users, all posts, and all tags currently in use"
```
The agent will: call Users.List, Posts.List, and ListTags (potentially in parallel)

## How It Works

The key insight: **you don't need to write any agent-specific code**. The MCP gateway discovers services from the registry, extracts tool schemas from Go types, and generates descriptions from doc comments.

```go
service := micro.New("platform",
    micro.Address(":9090"),
    mcp.WithMCP(":3001"),  // This one line makes everything AI-accessible
)

service.Handle(users)
service.Handle(posts)
service.Handle(&CommentService{})
service.Handle(&MailService{})
```

Each handler method becomes an MCP tool. The `@example` tags in doc comments give agents sample inputs to learn from.

## Connecting to Claude Code

Add to your Claude Code MCP config:

```json
{
  "mcpServers": {
    "platform": {
      "command": "curl",
      "args": ["-s", "http://localhost:3001/mcp/tools"]
    }
  }
}
```

Or use stdio transport:

```bash
micro mcp serve --registry mdns
```

## Architecture

```
Agent (Claude, GPT, etc.)
    │
    ▼
MCP Gateway (:3001)         ← Discovers services, generates tools
    │
    ▼
Go Micro RPC (:9090)        ← Standard service mesh
    │
    ├── UserService          ← Signup, Login, Profile
    ├── PostService          ← CRUD + Tags
    ├── CommentService       ← Threaded comments
    └── MailService          ← Internal messaging
```

## Relation to micro/blog

This example is a simplified, self-contained version of [micro/blog](https://github.com/micro/blog). The real platform splits each service into its own binary with protobuf definitions. This example uses Go structs directly for simplicity, but the MCP integration works identically either way — the gateway discovers services from the registry regardless of how they're implemented.
