# CRUD Contact Book Example

A complete CRUD service with MCP integration — the kind of service you'd actually build in production.

## What This Shows

- **6 operations**: Create, Get, Update, Delete, List, Search
- **Rich documentation**: Every handler has doc comments with `@example` tags
- **Struct tag descriptions**: All fields have `description` tags for agents
- **Input validation**: Required field checks with clear error messages
- **Partial updates**: Update only changes non-empty fields
- **Seed data**: Starts with 3 contacts so agents can explore immediately

## Run

```bash
go run .
```

## Test

```bash
# List all MCP tools
curl http://localhost:3001/mcp/tools | jq

# Create a contact
curl -X POST http://localhost:3001/mcp/call \
  -H 'Content-Type: application/json' \
  -d '{"tool": "contacts.Contacts.Create", "arguments": {"name": "Dave", "email": "dave@example.com"}}'

# Search contacts
curl -X POST http://localhost:3001/mcp/call \
  -H 'Content-Type: application/json' \
  -d '{"tool": "contacts.Contacts.Search", "arguments": {"query": "engineer"}}'
```

## Use with Claude Code

```bash
micro mcp serve
```

Then ask: "List all contacts and find the engineers."

## Key Patterns

### Doc Comments for Agents

```go
// Create adds a new contact to the book. Name and email are required.
//
// @example {"name": "Dave Wilson", "email": "dave@example.com", "role": "Engineer"}
func (h *Contacts) Create(ctx context.Context, req *CreateRequest, rsp *CreateResponse) error {
```

### Struct Tag Descriptions

```go
type Contact struct {
    ID    string `json:"id" description:"Unique contact identifier"`
    Name  string `json:"name" description:"Full name"`
    Email string `json:"email" description:"Email address"`
}
```

### Partial Updates

Only update fields that are provided (non-empty), so agents can change one field without overwriting others:

```go
if req.Name != "" {
    contact.Name = req.Name
}
```
