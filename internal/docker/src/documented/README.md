# Documented Service Example

This example demonstrates how to document your go-micro service handlers so that AI agents can understand them better. The MCP gateway parses Go comments and struct tags to generate rich tool descriptions.

## Documentation Features

### 1. **Go Doc Comments**

Standard Go documentation comments are used as the tool description:

```go
// GetUser retrieves a user by ID from the database.
//
// This endpoint fetches a user's complete profile including their name,
// email, and age. If the user doesn't exist, an error is returned.
func (u *Users) GetUser(ctx context.Context, req *GetUserRequest, rsp *GetUserResponse) error {
    // ...
}
```

### 2. **JSDoc-Style Tags**

Use `@param`, `@return`, and `@example` tags for detailed documentation:

```go
// CreateUser creates a new user in the system.
//
// @param name {string} User's full name (required, 1-100 characters)
// @param email {string} User's email address (required, must be valid email format)
// @param age {number} User's age (optional, must be 0-150 if provided)
// @return {User} The newly created user with generated ID
// @example
//   {
//     "name": "Alice Smith",
//     "email": "alice@example.com",
//     "age": 30
//   }
func (u *Users) CreateUser(ctx context.Context, req *CreateUserRequest, rsp *CreateUserResponse) error {
    // ...
}
```

### 3. **Struct Tags**

Add `description` tags to struct fields for better schema:

```go
type User struct {
    ID    string `json:"id" description:"User's unique identifier (UUID format)"`
    Name  string `json:"name" description:"User's full name"`
    Email string `json:"email" description:"User's email address"`
    Age   int    `json:"age,omitempty" description:"User's age (optional)"`
}
```

## Running the Example

### 1. Start the Service

```bash
cd examples/mcp/documented
go run main.go
```

Output:
```
Users service starting...
Service: users
Endpoints:
  - Users.GetUser
  - Users.CreateUser
MCP Gateway: http://localhost:3000
```

### 2. Test MCP Tools

List available tools:
```bash
curl http://localhost:3000/mcp/tools | jq
```

You'll see rich descriptions:

```json
{
  "tools": [
    {
      "name": "users.Users.GetUser",
      "description": "GetUser retrieves a user by ID from the database",
      "inputSchema": {
        "type": "object",
        "properties": {
          "id": {
            "type": "string",
            "description": "User ID in UUID format (e.g., \"123e4567-e89b-12d3-a456-426614174000\")"
          }
        },
        "required": ["id"],
        "examples": [
          "{\"id\": \"123e4567-e89b-12d3-a456-426614174000\"}"
        ]
      }
    },
    {
      "name": "users.Users.CreateUser",
      "description": "CreateUser creates a new user in the system",
      "inputSchema": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string",
            "description": "User's full name (required, 1-100 characters)"
          },
          "email": {
            "type": "string",
            "description": "User's email address (required, must be valid email format)"
          },
          "age": {
            "type": "number",
            "description": "User's age (optional, must be 0-150 if provided)"
          }
        },
        "required": ["name", "email"],
        "examples": [
          "{\"name\": \"Alice Smith\", \"email\": \"alice@example.com\", \"age\": 30}"
        ]
      }
    }
  ]
}
```

### 3. Call a Tool

Get existing user:
```bash
curl -X POST http://localhost:3000/mcp/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "users.Users.GetUser",
    "input": {"id": "user-1"}
  }'
```

Create new user:
```bash
curl -X POST http://localhost:3000/mcp/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "users.Users.CreateUser",
    "input": {
      "name": "Alice Smith",
      "email": "alice@example.com",
      "age": 30
    }
  }'
```

### 4. Use with Claude Code

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "users-service": {
      "command": "go",
      "args": ["run", "/path/to/examples/mcp/documented/main.go"]
    }
  }
}
```

Then in Claude Code, ask:
```
> You: "Show me user-1's profile"

Claude will:
1. See the GetUser tool with rich description
2. Understand it needs an "id" parameter (UUID format)
3. Call users.Users.GetUser with {"id": "user-1"}
4. Return the user profile
```

## Documentation Best Practices

### DO: Write Clear Descriptions

```go
// ✅ Good: Clear, explains what and why
// GetUser retrieves a user by ID from the database.
// Returns full profile including email, name, and preferences.
```

```go
// ❌ Bad: Vague, no context
// Get gets a user
```

### DO: Specify Parameter Constraints

```go
// ✅ Good: Specifies format and constraints
// @param id {string} User ID in UUID format (e.g., "123e4567-e89b-12d3-a456-426614174000")
// @param age {number} User's age (must be 0-150)
```

```go
// ❌ Bad: No constraints or format
// @param id {string} The ID
```

### DO: Provide Examples

```go
// ✅ Good: Real example agents can use
// @example
//   {
//     "name": "Alice Smith",
//     "email": "alice@example.com",
//     "age": 30
//   }
```

```go
// ❌ Bad: No example
// (agents have to guess the format)
```

### DO: Use Descriptive Struct Tags

```go
// ✅ Good: Explains what the field is
type User struct {
    ID string `json:"id" description:"User's unique identifier (UUID format)"`
}
```

```go
// ❌ Bad: No description
type User struct {
    ID string `json:"id"`
}
```

## How It Works

1. **Go Doc Parsing**
   - The MCP gateway reads your service's Go comments
   - First line becomes the tool description
   - Full comment becomes the detailed description

2. **JSDoc Tag Parsing**
   - `@param` tags enhance parameter descriptions
   - `@return` tags describe what the tool returns
   - `@example` tags provide usage examples

3. **Struct Tag Reading**
   - `description` tags add context to fields
   - `json:"field,omitempty"` marks optional fields
   - Used to generate JSON Schema for parameters

4. **Schema Generation**
   - Combines parsed documentation with type information
   - Creates rich JSON Schema for each tool
   - Agents use this to understand how to call your service

## Impact on Agent Performance

### Without Documentation

```
Tool: users.Users.GetUser
Description: Call GetUser on users service
Parameters: { "id": "string" }
```

Agent thinks: *"What's an ID? What format? What if I pass the wrong thing?"*

### With Documentation

```
Tool: users.Users.GetUser
Description: Retrieves a user by ID from the database. Returns full profile
             including email, name, and preferences.
Parameters:
  - id (string, required): User ID in UUID format
    Example: "123e4567-e89b-12d3-a456-426614174000"
Example:
  {"id": "user-1"}
```

Agent thinks: *"I need a UUID format ID. I can use 'user-1' from the example!"*

**Result:** Agent calls your service correctly on the first try!

## Next Steps

- Document all your service handlers with clear descriptions
- Add `@param`, `@return`, and `@example` tags
- Use `description` tags in struct fields
- Test with Claude Code to see how agents understand your services

## License

Apache 2.0
