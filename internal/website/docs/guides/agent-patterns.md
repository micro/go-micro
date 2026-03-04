---
layout: default
---

# Agent Integration Patterns

This guide covers common patterns for integrating AI agents with Go Micro services, from single-agent workflows to multi-agent architectures.

## Pattern 1: Single Agent with Multiple Services

The simplest and most common pattern. One AI agent has access to multiple microservices as MCP tools.

```
User → AI Agent → MCP Gateway → [Service A, Service B, Service C]
```

### Setup

Run multiple services and expose them all through one MCP gateway:

```go
users := micro.New("users", micro.Address(":8081"))
tasks := micro.New("tasks", micro.Address(":8082"))
notifications := micro.New("notifications", micro.Address(":8083"))

// Run all together as a modular monolith
g := micro.NewGroup(users, tasks, notifications)
g.Run()
```

With `micro run`, all services are discovered automatically via the registry, and the MCP tools endpoint at `/api/mcp/tools` exposes every endpoint from every service.

### When to Use

- Most applications start here
- Agent needs to orchestrate across services (e.g., "create a task and notify the assignee")
- You want the agent to choose which service to call based on the user's request

## Pattern 2: Scoped Agents

Different agents have access to different subsets of tools via scopes.

```
Customer Agent  → MCP Gateway → [orders:read, support:write]
Internal Agent  → MCP Gateway → [orders:*, users:*, billing:*]
Admin Agent     → MCP Gateway → [*]
```

### Setup

Create tokens with different scopes for each agent:

```go
// Gateway with scope enforcement
mcp.ListenAndServe(":3000", mcp.Options{
    Registry: reg,
    Auth:     authProvider,
    Scopes: map[string][]string{
        "billing.Billing.Charge":    {"billing:admin"},
        "users.Users.Delete":        {"users:admin"},
        "orders.Orders.List":        {"orders:read"},
        "orders.Orders.Create":      {"orders:write"},
        "support.Support.CreateTicket": {"support:write"},
    },
})
```

Then issue different tokens:
- Customer-facing agent token: `scopes=["orders:read", "support:write"]`
- Internal agent token: `scopes=["orders:read", "orders:write", "users:read"]`
- Admin agent token: `scopes=["*"]`

### When to Use

- Different trust levels for different agents
- Customer-facing vs internal agents
- Compliance requirements (e.g., PCI, HIPAA)

## Pattern 3: Agent as Service Consumer

Your Go Micro service itself calls an AI model to process data, using the `model` package.

```
User → API → Your Service → AI Model (Claude/GPT)
                          → Other Services
```

### Setup

```go
import (
    "go-micro.dev/v5/model"
    _ "go-micro.dev/v5/model/anthropic"
)

type SummaryService struct {
    ai    model.Model
    tasks *TaskClient
}

func NewSummaryService() *SummaryService {
    return &SummaryService{
        ai: model.New("anthropic",
            model.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
            model.WithModel("claude-sonnet-4-20250514"),
        ),
    }
}

// Summarize generates an AI summary of a project's tasks.
// Returns a natural language summary of task status, blockers, and progress.
//
// @example {"project_id": "proj-1"}
func (s *SummaryService) Summarize(ctx context.Context, req *SummarizeRequest, rsp *SummarizeResponse) error {
    // Fetch tasks from another service
    tasks, err := s.tasks.List(ctx, req.ProjectID)
    if err != nil {
        return err
    }

    // Use AI to summarize
    resp, err := s.ai.Generate(ctx, &model.Request{
        Prompt:       fmt.Sprintf("Summarize these tasks:\n%s", formatTasks(tasks)),
        SystemPrompt: "You are a concise project manager. Summarize task status in 2-3 sentences.",
    })
    if err != nil {
        return err
    }

    rsp.Summary = resp.Reply
    return nil
}
```

### When to Use

- Your service needs to process natural language
- Generating summaries, classifications, or extractions
- Enriching data with AI before returning to the caller

## Pattern 4: Agent with Tool Calling

An AI model calls your services as tools, with automatic tool execution via the model package.

```
User → Your App → AI Model ←→ MCP Tools (your services)
```

### Setup

```go
import (
    "go-micro.dev/v5/model"
    _ "go-micro.dev/v5/model/anthropic"
)

// Define tools from your service endpoints
tools := []model.Tool{
    {
        Name:        "create_task",
        Description: "Create a new task with title and assignee",
        Properties: map[string]any{
            "title":    map[string]any{"type": "string", "description": "Task title"},
            "assignee": map[string]any{"type": "string", "description": "Username"},
        },
    },
    {
        Name:        "list_tasks",
        Description: "List tasks filtered by status",
        Properties: map[string]any{
            "status": map[string]any{"type": "string", "description": "Filter: todo, in_progress, done"},
        },
    },
}

// Handle tool calls by routing to your services
toolHandler := func(name string, input map[string]any) (any, string) {
    switch name {
    case "create_task":
        var rsp CreateResponse
        err := client.Call(ctx, "tasks", "TaskService.Create", input, &rsp)
        if err != nil {
            return nil, fmt.Sprintf(`{"error": "%s"}`, err)
        }
        b, _ := json.Marshal(rsp)
        return rsp, string(b)
    case "list_tasks":
        var rsp ListResponse
        err := client.Call(ctx, "tasks", "TaskService.List", input, &rsp)
        if err != nil {
            return nil, fmt.Sprintf(`{"error": "%s"}`, err)
        }
        b, _ := json.Marshal(rsp)
        return rsp, string(b)
    }
    return nil, `{"error": "unknown tool"}`
}

m := model.New("anthropic",
    model.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
    model.WithToolHandler(toolHandler),
)

// The model will automatically call tools and return the final answer
resp, err := m.Generate(ctx, &model.Request{
    Prompt:       "Create a task for Alice to review the PR and tell me what tasks she has",
    SystemPrompt: "You are a helpful project management assistant",
    Tools:        tools,
})

fmt.Println(resp.Answer)
// "I've created a task for Alice to review the PR. She now has 3 tasks: ..."
```

### When to Use

- Building a chatbot or assistant that manages your services
- The agent playground in `micro run` uses this pattern
- You want the AI to decide which tools to call and in what order

## Pattern 5: Event-Driven Agent Triggers

Services emit events that trigger agent actions via the broker.

```
Service → Broker Event → Agent Handler → AI Model → Action
```

### Setup

```go
// Publisher: emit events from your service
broker.Publish("tasks.created", &broker.Message{
    Body: taskJSON,
})

// Subscriber: agent handler reacts to events
broker.Subscribe("tasks.created", func(p broker.Event) error {
    var task Task
    json.Unmarshal(p.Message().Body, &task)

    // Use AI to auto-assign based on task content
    resp, err := ai.Generate(ctx, &model.Request{
        Prompt: fmt.Sprintf("Who should handle this task? Title: %s, Description: %s. Team: alice (frontend), bob (backend), charlie (devops)", task.Title, task.Description),
        SystemPrompt: "Reply with just the username of the best person to handle this task.",
    })

    // Auto-assign
    client.Call(ctx, "tasks", "TaskService.Update", map[string]any{
        "id": task.ID,
        "assignee": strings.TrimSpace(resp.Reply),
    }, nil)

    return nil
})
```

### When to Use

- Automated workflows triggered by service events
- AI-powered routing, classification, or triage
- Background processing without user interaction

## Pattern 6: Claude Code Integration

Developers use Claude Code with your services as MCP tools for local development workflows.

```
Developer → Claude Code → stdio MCP → [local services]
```

### Setup

```bash
# Start services locally
micro run

# In another terminal, use Claude Code with your services
# Claude Code config (~/.claude/claude_desktop_config.json):
```

```json
{
  "mcpServers": {
    "my-project": {
      "command": "micro",
      "args": ["mcp", "serve"]
    }
  }
}
```

Now in Claude Code:

```
"List all tasks that are blocked"
"Create a user account for the new hire"
"Check the health of all services"
```

### When to Use

- Developer productivity workflows
- Managing services during development
- Testing and debugging with natural language

## Pattern 7: LangChain / LlamaIndex Integration

Use the official Python SDKs to connect agent frameworks directly to your services.

### LangChain

```python
from langchain_go_micro import GoMicroToolkit

# Connect to MCP gateway
toolkit = GoMicroToolkit(
    base_url="http://localhost:3000",
    token="Bearer <token>",
)

# Get LangChain tools automatically
tools = toolkit.get_tools()

# Use with any LangChain agent
from langchain.agents import AgentExecutor, create_tool_calling_agent
agent = create_tool_calling_agent(llm, tools, prompt)
executor = AgentExecutor(agent=agent, tools=tools)
executor.invoke({"input": "Create a task for Alice"})
```

### LlamaIndex

```python
from go_micro_llamaindex import GoMicroToolkit

toolkit = GoMicroToolkit(
    base_url="http://localhost:3000",
    token="Bearer <token>",
)

# Use as LlamaIndex tools
tools = toolkit.to_tool_list()

# Use with a LlamaIndex agent
from llama_index.core.agent import ReActAgent
agent = ReActAgent.from_tools(tools, llm=llm)
agent.chat("What tasks are assigned to Bob?")
```

### When to Use

- Python-based agent pipelines
- RAG (Retrieval-Augmented Generation) workflows with LlamaIndex
- Multi-step LangChain chains that orchestrate your services
- Teams that prefer Python for AI/ML work

## Pattern 8: Standalone Gateway for Production

Run the MCP gateway as a separate, horizontally scalable process.

```
                    ┌──────────────────┐
Claude/GPT/Agent ──→│ micro-mcp-gateway │──→ Service A (consul)
                    │   (standalone)    │──→ Service B (consul)
                    └──────────────────┘──→ Service C (consul)
```

### Setup

```bash
micro-mcp-gateway \
  --registry consul \
  --registry-address consul:8500 \
  --address :3000 \
  --auth jwt \
  --rate-limit 10 \
  --rate-burst 20 \
  --audit
```

Or via Docker:

```bash
docker run -p 3000:3000 ghcr.io/micro/micro-mcp-gateway \
  --registry consul \
  --registry-address consul:8500
```

### When to Use

- Production deployments where you want the gateway to scale independently
- Multiple teams deploying services but sharing one MCP endpoint
- Enterprise environments needing centralized auth and audit

## Choosing a Pattern

| Pattern | Complexity | Best For |
|---------|-----------|----------|
| Single Agent | Low | Most applications, getting started |
| Scoped Agents | Medium | Multi-tenant, compliance |
| Agent as Consumer | Medium | AI-enhanced services |
| Tool Calling | Medium | Chatbots, assistants |
| Event-Driven | High | Automation, background processing |
| Claude Code | Low | Developer workflows |
| LangChain/LlamaIndex | Medium | Python agent pipelines, RAG |
| Standalone Gateway | Medium | Production, enterprise |

Start with **Pattern 1** (single agent) and add complexity as needed. Most applications don't need multi-agent architectures.

## Anti-Patterns

### Don't: Chain Agents Without Coordination

```
Agent A → Agent B → Agent C  (no shared state, no trace IDs)
```

Instead, use a single agent with multiple tools, or share trace IDs via metadata.

### Don't: Give Agents Unrestricted Access

```
Customer Agent → scopes=["*"]  (dangerous!)
```

Always use the minimum required scopes. See the [MCP Security Guide](mcp-security.md).

### Don't: Skip Error Documentation

If agents don't know what errors are possible, they can't handle them gracefully. Always document error cases in your handler comments.

### Don't: Build Agent Logic into Services

Keep services as pure business logic. Let the agent (or the agent framework) handle orchestration, retries, and decision-making. Your service should just do one thing well.

## Next Steps

- [Building AI-Native Services](ai-native-services.md) - End-to-end tutorial
- [MCP Security Guide](mcp-security.md) - Auth and scopes
- [Tool Description Best Practices](tool-descriptions.md) - Better docs for agents
- [Model Package](../../model/README.md) - AI provider interface
