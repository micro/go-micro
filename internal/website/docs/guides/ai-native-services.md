---
layout: default
---

# Building AI-Native Services

This guide walks you through building a Go Micro service that is AI-native from the start — meaning AI agents can discover, understand, and call your service automatically via the Model Context Protocol (MCP).

## What You'll Build

A **task management service** with full CRUD operations that:
- Exposes every endpoint as an MCP tool automatically
- Has rich documentation that agents can read
- Includes auth scopes for write operations
- Works with Claude Code, the agent playground, and any MCP client

## Prerequisites

```bash
go install go-micro.dev/v5/cmd/micro@v5.16.0
```

## Step 1: Create the Service

```bash
micro new tasks
cd tasks
```

## Step 2: Define Your Types

Design your request/response types with `description` tags. These tags become parameter descriptions that agents read:

```go
package main

import "context"

// Request types with description tags for AI agents
type Task struct {
    ID          string `json:"id" description:"Unique task identifier"`
    Title       string `json:"title" description:"Short task title (max 100 chars)"`
    Description string `json:"description" description:"Detailed task description"`
    Status      string `json:"status" description:"Task status: todo, in_progress, or done"`
    Assignee    string `json:"assignee,omitempty" description:"Username of assigned person"`
}

type CreateRequest struct {
    Title       string `json:"title" description:"Task title (required, max 100 chars)"`
    Description string `json:"description" description:"Detailed description of the task"`
    Assignee    string `json:"assignee,omitempty" description:"Username to assign the task to"`
}

type CreateResponse struct {
    Task *Task `json:"task" description:"The newly created task"`
}

type GetRequest struct {
    ID string `json:"id" description:"Task ID to retrieve"`
}

type GetResponse struct {
    Task *Task `json:"task" description:"The requested task"`
}

type ListRequest struct {
    Status string `json:"status,omitempty" description:"Filter by status: todo, in_progress, done (optional)"`
}

type ListResponse struct {
    Tasks []*Task `json:"tasks" description:"List of matching tasks"`
}

type UpdateRequest struct {
    ID     string `json:"id" description:"Task ID to update"`
    Status string `json:"status" description:"New status: todo, in_progress, or done"`
}

type UpdateResponse struct {
    Task *Task `json:"task" description:"The updated task"`
}

type DeleteRequest struct {
    ID string `json:"id" description:"Task ID to delete"`
}

type DeleteResponse struct {
    Deleted bool `json:"deleted" description:"True if the task was deleted"`
}
```

**Key point:** The `description` tags are parsed by the MCP gateway and shown to agents as parameter documentation. Be specific about formats, constraints, and valid values.

## Step 3: Write the Handler with Doc Comments

Write standard Go doc comments on every handler method. The MCP gateway extracts these automatically at registration time.

```go
type TaskService struct {
    tasks map[string]*Task
    nextID int
}

// Create creates a new task with the given title and description.
// Returns the created task with a generated ID and initial status of "todo".
//
// @example {"title": "Fix login bug", "description": "Users can't log in with SSO", "assignee": "alice"}
func (t *TaskService) Create(ctx context.Context, req *CreateRequest, rsp *CreateResponse) error {
    t.nextID++
    task := &Task{
        ID:          fmt.Sprintf("task-%d", t.nextID),
        Title:       req.Title,
        Description: req.Description,
        Status:      "todo",
        Assignee:    req.Assignee,
    }
    t.tasks[task.ID] = task
    rsp.Task = task
    return nil
}

// Get retrieves a task by its unique ID.
// Returns an error if the task does not exist.
//
// @example {"id": "task-1"}
func (t *TaskService) Get(ctx context.Context, req *GetRequest, rsp *GetResponse) error {
    task, ok := t.tasks[req.ID]
    if !ok {
        return fmt.Errorf("task %s not found", req.ID)
    }
    rsp.Task = task
    return nil
}

// List returns all tasks, optionally filtered by status.
// If no status filter is provided, returns all tasks.
// Valid status values: "todo", "in_progress", "done".
//
// @example {"status": "todo"}
func (t *TaskService) List(ctx context.Context, req *ListRequest, rsp *ListResponse) error {
    for _, task := range t.tasks {
        if req.Status == "" || task.Status == req.Status {
            rsp.Tasks = append(rsp.Tasks, task)
        }
    }
    return nil
}

// Update changes the status of an existing task.
// Valid status transitions: todo -> in_progress -> done.
// Returns an error if the task does not exist.
//
// @example {"id": "task-1", "status": "in_progress"}
func (t *TaskService) Update(ctx context.Context, req *UpdateRequest, rsp *UpdateResponse) error {
    task, ok := t.tasks[req.ID]
    if !ok {
        return fmt.Errorf("task %s not found", req.ID)
    }
    task.Status = req.Status
    rsp.Task = task
    return nil
}

// Delete removes a task by ID. This action is irreversible.
// Returns an error if the task does not exist.
//
// @example {"id": "task-1"}
func (t *TaskService) Delete(ctx context.Context, req *DeleteRequest, rsp *DeleteResponse) error {
    if _, ok := t.tasks[req.ID]; !ok {
        return fmt.Errorf("task %s not found", req.ID)
    }
    delete(t.tasks, req.ID)
    rsp.Deleted = true
    return nil
}
```

**What agents see:** Each method's doc comment becomes the tool description. The `@example` tag provides a valid JSON input that agents can reference.

## Step 4: Register with Scopes

Use `server.WithEndpointScopes()` to control which agents can call which endpoints:

```go
package main

import (
    "context"
    "fmt"

    "go-micro.dev/v5"
    "go-micro.dev/v5/server"
)

func main() {
    service := micro.NewService(
        micro.Name("tasks"),
        micro.Address(":8081"),
    )
    service.Init()

    handler := service.Server().NewHandler(
        &TaskService{tasks: make(map[string]*Task)},
        // Read operations: any authenticated agent
        server.WithEndpointScopes("TaskService.Get", "tasks:read"),
        server.WithEndpointScopes("TaskService.List", "tasks:read"),
        // Write operations: agents with write scope
        server.WithEndpointScopes("TaskService.Create", "tasks:write"),
        server.WithEndpointScopes("TaskService.Update", "tasks:write"),
        // Delete: admin only
        server.WithEndpointScopes("TaskService.Delete", "tasks:admin"),
    )
    service.Server().Handle(handler)

    service.Run()
}
```

## Step 5: Run with MCP

```bash
micro run
```

Your service is now available at:
- **Web Dashboard:** http://localhost:8080/
- **Agent Playground:** http://localhost:8080/agent
- **MCP Tools:** http://localhost:8080/api/mcp/tools
- **API Gateway:** http://localhost:8080/api/tasks/TaskService/Create

### Use with Claude Code

```bash
# Start MCP server for Claude Code
micro mcp serve
```

Add to your Claude Code config:

```json
{
  "mcpServers": {
    "tasks": {
      "command": "micro",
      "args": ["mcp", "serve"]
    }
  }
}
```

Now Claude can manage your tasks:

```
You: "Create a task to fix the login bug and assign it to alice"
Claude: [calls tasks.TaskService.Create with {"title": "Fix login bug", ...}]
        Created task-1: "Fix login bug" assigned to alice.

You: "What tasks does alice have?"
Claude: [calls tasks.TaskService.List]
        Alice has 1 task: "Fix login bug" (status: todo)

You: "Mark it as in progress"
Claude: [calls tasks.TaskService.Update with {"id": "task-1", "status": "in_progress"}]
        Updated task-1 to "in_progress".
```

## Step 6: Test Your Tools

Use the CLI to verify tools work:

```bash
# List all available tools
micro mcp list

# Test a specific tool
micro mcp test tasks.TaskService.Create

# Generate documentation
micro mcp docs

# Export for LangChain
micro mcp export --format langchain
```

## Step 7: Use the Model Package (Optional)

If your service needs to call AI models directly:

```go
import (
    "go-micro.dev/v5/model"
    _ "go-micro.dev/v5/model/anthropic"
)

m := model.New("anthropic",
    model.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
)

resp, err := m.Generate(ctx, &model.Request{
    Prompt:       "Summarize these tasks: " + taskJSON,
    SystemPrompt: "You are a project manager assistant",
})
```

## Checklist

Before shipping an AI-native service:

- [ ] Every handler method has a doc comment explaining what it does
- [ ] Every method has an `@example` tag with realistic JSON input
- [ ] Request struct fields have `description` tags
- [ ] Write/delete operations have auth scopes
- [ ] You've tested with `micro mcp test` to verify tools work
- [ ] You've tested with Claude Code or the agent playground

## What Happens Under the Hood

```
1. You write Go comments on handler methods
2. micro registers the handler and extracts docs via go/ast
3. Docs are stored in the service registry as endpoint metadata
4. MCP gateway discovers services via the registry
5. Gateway generates JSON Schema tools with descriptions
6. AI agents query the tools endpoint and see rich descriptions
7. Agents call tools via JSON-RPC, gateway routes to your handler
```

## Next Steps

- [MCP Security Guide](mcp-security.md) - Configure auth and scopes for production
- [Tool Description Best Practices](tool-descriptions.md) - Write comments that make agents smarter
- [Agent Integration Patterns](agent-patterns.md) - Multi-agent workflows
- [MCP Documentation](../mcp.md) - Full MCP reference
