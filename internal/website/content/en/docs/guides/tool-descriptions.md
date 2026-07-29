---
layout: default
---

# Best Practices for Tool Descriptions

Your Go doc comments become the documentation that AI agents read when deciding how to call your service. Better descriptions lead to fewer errors, faster task completion, and a better user experience.

## How Agents Use Your Docs

When an AI agent receives a user request like "create a task for Alice", it:

1. Queries the MCP tools endpoint for available tools
2. Reads each tool's **description** to understand what it does
3. Reads the **parameter schema** and descriptions to build the input
4. References the **example** to verify the format
5. Makes the call

If any of these are missing or unclear, the agent guesses — and often guesses wrong.

## The Three Essentials

Every handler method needs three things:

### 1. A Clear Description (Doc Comment)

```go
// Create creates a new task with the given title and description.
// Returns the created task with a generated ID and initial status of "todo".
// The assignee field is optional; if omitted, the task is unassigned.
```

**Rules:**
- First sentence: what the method does (imperative mood)
- Second sentence: what it returns
- Additional sentences: important behavior, constraints, edge cases

### 2. An Example Input (`@example`)

```go
// @example {"title": "Fix login bug", "description": "Users can't log in with SSO", "assignee": "alice"}
```

**Rules:**
- Use realistic values, not placeholders like `"string"` or `"test"`
- Include all required fields
- Include at least one optional field to show the format
- Keep it on one line (the parser reads until end of line)

### 3. Field Descriptions (`description` tag)

```go
type CreateRequest struct {
    Title    string `json:"title" description:"Task title (required, max 100 chars)"`
    Assignee string `json:"assignee,omitempty" description:"Username to assign (optional)"`
}
```

**Rules:**
- State the type constraint if not obvious (e.g., "UUID format", "ISO 8601 date")
- List valid values for enums (e.g., "todo, in_progress, or done")
- Note if optional (matches `omitempty`)

## Good vs Bad Examples

### Describing What a Method Does

**Good:**
```go
// GetUser retrieves a user by their unique ID from the database.
// Returns the full profile including name, email, and preferences.
// Returns an error if the user does not exist.
//
// @example {"id": "user-123"}
func (s *UserService) GetUser(ctx context.Context, req *GetRequest, rsp *GetResponse) error {
```

**Bad:**
```go
// Gets user
func (s *UserService) GetUser(ctx context.Context, req *GetRequest, rsp *GetResponse) error {
```

The bad version forces the agent to guess what "gets user" means, what parameters are needed, and what format the ID takes.

### Describing Parameters

**Good:**
```go
type SearchRequest struct {
    Query   string `json:"query" description:"Search query string (min 2 chars, max 200)"`
    Page    int    `json:"page,omitempty" description:"Page number, starting from 1 (default: 1)"`
    PerPage int    `json:"per_page,omitempty" description:"Results per page, 1-100 (default: 20)"`
    SortBy  string `json:"sort_by,omitempty" description:"Sort field: relevance, date, or name (default: relevance)"`
}
```

**Bad:**
```go
type SearchRequest struct {
    Q string `json:"q"`
    P int    `json:"p"`
    N int    `json:"n"`
    S string `json:"s"`
}
```

### Providing Examples

**Good:**
```go
// @example {"query": "microservices architecture", "page": 1, "per_page": 10, "sort_by": "relevance"}
```

**Bad:**
```go
// @example {"q": "string", "p": 0, "n": 0}
```

## Patterns for Common Scenarios

### CRUD Operations

```go
// Create creates a new [resource].
// Returns the created [resource] with a generated ID.
//
// @example {realistic create payload}

// Get retrieves a [resource] by ID.
// Returns an error if the [resource] does not exist.
//
// @example {"id": "realistic-id"}

// List returns all [resources], optionally filtered by [criteria].
// Returns an empty list if no [resources] match.
//
// @example {"status": "active"}

// Update modifies an existing [resource].
// Only the provided fields are updated; omitted fields are unchanged.
// Returns an error if the [resource] does not exist.
//
// @example {"id": "realistic-id", "field": "new-value"}

// Delete removes a [resource] by ID. This action is irreversible.
// Returns an error if the [resource] does not exist.
//
// @example {"id": "realistic-id"}
```

### Search Endpoints

```go
// Search finds [resources] matching the query string.
// Supports full-text search across [fields].
// Results are paginated; use page and per_page to control pagination.
// Returns results sorted by relevance by default.
//
// @example {"query": "realistic search term", "page": 1, "per_page": 20}
```

### Actions with Side Effects

```go
// SendEmail sends an email notification to the specified recipient.
// This triggers an actual email delivery — use with caution.
// Returns an error if the email address is invalid or the mail server is unavailable.
//
// @example {"to": "alice@example.com", "subject": "Task assigned", "body": "You have a new task."}
```

### Methods with Complex Inputs

```go
// CreateReport generates a report for the specified date range and metrics.
// Processing may take up to 30 seconds for large date ranges.
// Valid metrics: cpu_usage, memory_usage, request_count, error_rate.
// Date format: YYYY-MM-DD (e.g., "2026-01-15").
//
// @example {"start_date": "2026-01-01", "end_date": "2026-01-31", "metrics": ["cpu_usage", "error_rate"]}
```

## Impact on Agent Performance

| Documentation Quality | First-Call Success Rate | Avg Calls to Complete |
|----------------------|------------------------|----------------------|
| No docs | ~25% | 3-4 calls |
| Basic (name only) | ~50% | 2-3 calls |
| Good (description + types) | ~80% | 1-2 calls |
| Excellent (description + types + example) | ~95% | 1 call |

## Testing Your Descriptions

### 1. Use `micro mcp list`

Check what agents will see:

```bash
micro mcp list
```

Verify each tool has a description and the schema looks correct.

### 2. Use `micro mcp docs`

Generate the full documentation:

```bash
micro mcp docs
```

Read through it as if you were an AI agent. Does it make sense without seeing the code?

### 3. Test with Claude Code

The ultimate test — add your service to Claude Code and try natural language commands:

```
"Create a task for Alice to fix the login bug"
"What tasks are assigned to Bob?"
"Mark task-1 as done"
```

If Claude gets it right on the first try, your docs are good.

### 4. Use `micro mcp test`

Test individual tools with specific inputs:

```bash
micro mcp test tasks.TaskService.Create
```

## Manual Overrides

If you can't modify the source code (e.g., third-party services), override descriptions at handler registration:

```go
handler := service.Server().NewHandler(
    new(LegacyService),
    server.WithEndpointDocs("LegacyService.Process", server.EndpointDocs{
        Description: "Process a payment transaction. Charges the specified amount to the customer's payment method on file.",
        Example:     `{"customer_id": "cust-123", "amount_cents": 4999, "currency": "USD"}`,
    }),
)
```

Manual docs take precedence over auto-extracted comments. This is useful for:
- Third-party or generated code where you can't add comments
- Overriding auto-extracted descriptions that aren't agent-friendly
- Adding examples to legacy endpoints

## Export Formats

You can export tool descriptions in different formats for use with agent frameworks:

```bash
# Human-readable documentation
micro mcp docs

# JSON for custom tooling
micro mcp export --format json

# LangChain Python format
micro mcp export --format langchain

# OpenAPI specification
micro mcp export --format openapi
```

## Common Mistakes

1. **Placeholder examples** — Using `"string"` or `"test"` instead of realistic values
2. **Missing enum values** — Not listing valid options for status/type fields
3. **Ambiguous field names** — Single-letter or abbreviated field names without descriptions
4. **No error documentation** — Not telling agents what can go wrong
5. **Missing optional field markers** — Not using `omitempty` or noting "(optional)"
6. **Overly technical descriptions** — Writing for Go developers instead of AI agents

## Next Steps

- [Building AI-Native Services](ai-native-services.md) - Full tutorial
- [MCP Security Guide](mcp-security.md) - Auth and scopes for production
- [Agent Integration Patterns](agent-patterns.md) - Multi-agent workflows
- [MCP Documentation Reference](https://github.com/micro/go-micro/blob/master/gateway/mcp/DOCUMENTATION.md) - Full API docs
