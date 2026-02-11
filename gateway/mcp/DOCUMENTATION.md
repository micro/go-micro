# MCP Tool Documentation

This document explains how to document your go-micro services so that AI agents can understand them better.

## Overview

The MCP gateway automatically exposes your microservices as tools that AI agents (like Claude) can call. By adding proper documentation to your service handlers, you help agents understand:

- **What the tool does** - The purpose and behavior
- **What parameters it needs** - Types, formats, constraints
- **What it returns** - Response structure and meaning
- **How to use it** - Example inputs and outputs

## Documentation Methods

go-micro **automatically extracts documentation** from your Go doc comments at registration time. You don't need to write any extra code!

### 1. Go Doc Comments (Automatic - Recommended)

Just write standard Go documentation comments on your handler methods:

```go
// GetUser retrieves a user by ID from the database. Returns full profile including email, name, and preferences.
//
// @example {"id": "user-1"}
func (s *UserService) GetUser(ctx context.Context, req *GetUserRequest, rsp *GetUserResponse) error {
    // implementation
}
```

When you register the handler, go-micro automatically:
- Extracts the doc comment as the tool description
- Parses the `@example` tag for example inputs
- Registers everything in the service registry
- Makes it available to the MCP gateway

**Supported Tags:**
- `@example <json>` - Example JSON input (highly recommended for AI agents)

**That's it!** No extra registration code needed:

```go
// Documentation is extracted automatically from method comments
handler := service.Server().NewHandler(new(UserService))
service.Server().Handle(handler)
```

### 2. Manual Registration (Optional Override)

For more control or to override auto-extracted docs, use `server.WithEndpointDocs()`:

```go
handler := service.Server().NewHandler(
    new(UserService),
    server.WithEndpointDocs(map[string]server.EndpointDoc{
        "UserService.GetUser": {
            Description: "Custom description that overrides the comment",
            Example:     `{"id": "user-123"}`,
        },
    }),
)
```

Manual metadata **takes precedence** over auto-extracted comments.

### 3. Endpoint Scopes (Auth)

Use `server.WithEndpointScopes()` to declare the auth scopes required for each
endpoint. The MCP gateway reads these from the registry and enforces them when
an `Auth` provider is configured.

```go
handler := service.Server().NewHandler(
    new(BlogService),
    server.WithEndpointScopes("Blog.Create", "blog:write"),
    server.WithEndpointScopes("Blog.Delete", "blog:write", "blog:admin"),
    server.WithEndpointScopes("Blog.Read", "blog:read"),
)
```

Scopes are stored as comma-separated values in endpoint metadata (`"scopes"` key)
and are propagated through the service registry just like descriptions and examples.

#### Gateway-Level Scope Overrides

An operator can also define or override scopes at the MCP gateway without
modifying individual services. This is useful for centralized policy management:

```go
mcp.Serve(mcp.Options{
    Registry: reg,
    Auth:     authProvider,
    Scopes: map[string][]string{
        "blog.Blog.Create": {"blog:write"},
        "blog.Blog.Delete": {"blog:admin"},
    },
})
```

Gateway-level scopes **take precedence** over service-level scopes.

### 4. Struct Tags (For Field Descriptions)

Add descriptions to struct fields using the `description` tag:

```go
type User struct {
    ID    string `json:"id" description:"User's unique identifier (UUID format)"`
    Name  string `json:"name" description:"User's full name"`
    Email string `json:"email" description:"User's email address"`
    Age   int    `json:"age,omitempty" description:"User's age (optional)"`
}
```

The `description` tag is used to generate parameter descriptions in the JSON Schema.

## How It Works

### Automatic Extraction Pipeline

```
1. Handler Registration (Your Service)
   ├─> You write Go doc comments on methods
   ├─> Call service.Server().NewHandler(yourHandler)
   └─> go-micro automatically parses source files using go/ast

2. Documentation Extraction (Automatic)
   ├─> Read Go doc comments from handler method source
   ├─> Parse @example tags for sample inputs
   ├─> Extract struct tag descriptions
   └─> Merge with any manual metadata (manual wins)

3. Service Registry
   ├─> Store endpoint metadata in registry.Endpoint.Metadata
   ├─> Metadata distributed with service information
   └─> Available to all components (gateway, discovery, etc.)

4. MCP Gateway Discovery
   ├─> Query registry for services and endpoints
   ├─> Read description and example from endpoint.Metadata
   └─> Generate JSON Schema with documentation

5. Tool Creation
   └─> Create MCP tool with rich description for AI agents
```

### Example Output

For a documented handler, the MCP gateway generates:

```json
{
  "name": "users.UserService.GetUser",
  "description": "GetUser retrieves a user by ID from the database. Returns full profile including email, name, and preferences.",
  "inputSchema": {
    "type": "object",
    "description": "This endpoint fetches a user's complete profile...",
    "properties": {
      "id": {
        "type": "string",
        "description": "User ID in UUID format (e.g., \"123e4567-e89b-12d3-a456-426614174000\")"
      }
    },
    "required": ["id"],
    "examples": [
      "{\"id\": \"user-1\"}"
    ]
  }
}
```

## Best Practices

### Write for AI, Not Just Humans

AI agents parse your documentation literally. Be explicit:

**✅ Good:**
```go
// GetUser retrieves a user by their unique ID from the database.
// Returns the user's full profile including name, email, and preferences.
// If the user doesn't exist, returns an error with status 404.
//
// @param id {string} User ID in UUID v4 format (e.g., "123e4567-e89b-12d3-a456-426614174000")
// @return {User} User object with all profile fields populated
```

**❌ Bad:**
```go
// Gets a user
func GetUser(...) // No details, no context
```

### Specify Formats and Constraints

Tell agents exactly what format you expect:

**✅ Good:**
```go
// @param email {string} Email address in RFC 5322 format (must contain @ and domain)
// @param age {number} User's age (integer between 0-150)
// @param phone {string} Phone number in E.164 format (e.g., "+14155552671")
```

**❌ Bad:**
```go
// @param email {string} The email
// @param age {number} Age
```

### Provide Real Examples

Show agents actual valid inputs:

**✅ Good:**
```go
// @example
//   {
//     "name": "Alice Smith",
//     "email": "alice@example.com",
//     "age": 30,
//     "phone": "+14155552671"
//   }
```

**❌ Bad:**
```go
// @example
//   {
//     "name": "string",
//     "email": "string"
//   }
```

### Document Error Cases

Tell agents what can go wrong:

```go
// GetUser retrieves a user by ID.
//
// Returns error if:
// - User ID is not a valid UUID
// - User does not exist (404)
// - Database is unavailable (503)
//
// @param id {string} User ID in UUID format
```

### Use Descriptive Names

Field names should be self-explanatory:

**✅ Good:**
```go
type CreateUserRequest struct {
    FullName      string `json:"full_name" description:"User's complete name"`
    EmailAddress  string `json:"email_address" description:"Primary email for contact"`
    DateOfBirth   string `json:"date_of_birth" description:"Birth date in YYYY-MM-DD format"`
}
```

**❌ Bad:**
```go
type CreateUserRequest struct {
    N string `json:"n"` // What is n?
    E string `json:"e"` // What is e?
    D string `json:"d"` // What is d?
}
```

## Impact on Agent Performance

### Without Documentation

```
Agent: "I need to call GetUser but I don't know what format the ID should be.
        Is it a number? A string? A UUID? Let me try..."

❌ Calls with: {"id": 123}
❌ Calls with: {"id": "user123"}
❌ Calls with: {"id": "abc"}
✅ Calls with: {"id": "550e8400-e29b-41d4-a716-446655440000"} (after 4 attempts)
```

### With Documentation

```
Agent: "GetUser needs an ID in UUID format. The example shows the format.
        I'll use a valid UUID."

✅ Calls with: {"id": "550e8400-e29b-41d4-a716-446655440000"} (first attempt)
```

**Result:**
- **75% fewer failed calls**
- **Faster task completion**
- **Better user experience**

## Parser Implementation

The MCP gateway uses several parsers:

### 1. Go Doc Parser (`parseServiceDocs`)
- Extracts godoc comments from handler methods
- Parses JSDoc-style tags
- Returns `ToolDescription` struct

### 2. Struct Tag Parser (`ParseStructTags`)
- Reads `description` tags from struct fields
- Generates JSON Schema with field descriptions
- Marks required vs optional fields (omitempty)

### 3. Comment Parser (`ParseGoDocComment`)
- Regex-based extraction of @param, @return, @example tags
- Splits summary from detailed description
- Builds structured documentation

### 4. Type Mapper (`reflectTypeToJSONType`)
- Converts Go types to JSON Schema types
- Handles: string, int, float, bool, array, object
- Used for automatic schema generation

## Examples

See complete examples in:
- `examples/mcp/documented/` - Fully documented service
- `examples/auth/` - Auth service with documentation
- `examples/hello-world/` - Basic service

## Testing Documentation

### 1. List Tools

```bash
curl http://localhost:3000/mcp/tools | jq '.tools[0]'
```

Verify the description and schema are correct.

### 2. Use with Claude Code

Add to your Claude Code config and ask Claude to use your service. Claude will show you how it interprets your documentation.

### 3. Check Examples Work

Try the examples from your `@example` tags:

```bash
curl -X POST http://localhost:3000/mcp/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "users.UserService.GetUser",
    "input": <your-example-json>
  }'
```

## Future Enhancements

Planned improvements:

- [ ] Auto-extract examples from test files
- [ ] Validate documentation completeness (lint)
- [ ] Generate documentation from OpenAPI specs
- [ ] Support custom validation rules in tags
- [ ] Interactive documentation editor

## FAQ

**Q: Do I need to document every field?**
A: Document fields that are ambiguous or have constraints. Self-explanatory fields can rely on the field name.

**Q: Will this slow down my service?**
A: No. Documentation is parsed once at startup when the MCP gateway discovers services.

**Q: Can I use OpenAPI/Swagger specs instead?**
A: Not yet, but it's planned. For now, use Go comments and struct tags.

**Q: What if I don't document my handlers?**
A: The MCP gateway will still work, generating basic descriptions from method names and types. But agents will perform better with documentation.

**Q: How do I know if my documentation is good?**
A: Test it with Claude Code. If Claude understands your service and calls it correctly on the first try, your documentation is good!

**Q: How do I add auth scopes to my endpoints?**
A: Use `server.WithEndpointScopes()` when registering your handler:

```go
handler := service.Server().NewHandler(
    new(MyService),
    server.WithEndpointScopes("MyService.Create", "write"),
)
```

Or define scopes at the gateway level using `Scopes` in `mcp.Options`.

**Q: Can I set scopes at the gateway without changing services?**
A: Yes. Use the `Scopes` option on `mcp.Options` to define or override scopes for any tool at the gateway layer. This is useful for centralized policy management.

## License

Apache 2.0
