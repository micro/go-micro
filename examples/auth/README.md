# Auth Example

This example demonstrates how to use the auth wrappers to protect your microservices with authentication and authorization.

## Overview

The example includes:

- **Server** - A Greeter service with:
  - Protected endpoint: `Greeter.Hello` (requires auth)
  - Public endpoint: `Greeter.Health` (no auth required)

- **Client** - Makes calls to the server:
  - With authentication (successful)
  - Without authentication (fails as expected)

## Architecture

```
┌─────────────────────────────────────────┐
│            Client                       │
│  ┌────────────────────────────────┐    │
│  │  AuthClient Wrapper            │    │
│  │  - Adds Bearer token           │    │
│  │  - To all requests             │    │
│  └────────────────────────────────┘    │
└──────────────┬──────────────────────────┘
               │ RPC with Authorization: Bearer <token>
               │
               ▼
┌─────────────────────────────────────────┐
│            Server                       │
│  ┌────────────────────────────────┐    │
│  │  AuthHandler Wrapper           │    │
│  │  - Extracts token              │    │
│  │  - Verifies with auth.Inspect()│    │
│  │  - Checks with rules.Verify()  │    │
│  │  - Returns 401/403 if denied   │    │
│  └────────────────────────────────┘    │
│               │                         │
│               ▼                         │
│  ┌────────────────────────────────┐    │
│  │  Handler (Greeter.Hello)       │    │
│  │  - Gets account from context   │    │
│  │  - Processes request           │    │
│  └────────────────────────────────┘    │
└─────────────────────────────────────────┘
```

## Files

```
examples/auth/
├── README.md              # This file
├── proto/
│   ├── greeter.proto     # Service definition
│   └── greeter.pb.go     # Generated Go code
├── server/
│   └── main.go           # Protected service
└── client/
    └── main.go           # Client with auth
```

## Running the Example

### 1. Start the Server

```bash
cd server
go run main.go
```

The server will:
- Start the Greeter service
- Apply auth wrapper to protect endpoints
- Generate a test token and print it

Output:
```
=== Test Token Generated ===
Use this token to test the client:
TOKEN=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9... go run client/main.go

2026/02/11 10:00:00 Server [greeter] Listening on [::]:54321
```

### 2. Run the Client (With Auth)

In a new terminal:

```bash
cd client
TOKEN=<token-from-server> go run main.go
```

Output:
```
=== Test 1: Protected endpoint WITH auth ===
Response: Hello, test-user!

=== Test 2: Public endpoint (no auth needed) ===
Health Status: ok

=== Test 3: Protected endpoint WITHOUT auth (should fail) ===
Expected error: {"id":"greeter","code":401,"detail":"missing authorization token","status":"Unauthorized"}
```

### 3. Run the Client (Without Auth)

```bash
cd client
go run main.go
```

This will auto-generate a token for testing.

## Code Walkthrough

### Server Setup

```go
// 1. Create auth provider (JWT in production, noop for testing)
authProvider, err := jwt.NewAuth(
    auth.Issuer("go-micro"),
)

// 2. Create authorization rules
rules := auth.NewRules()
rules.Grant(&auth.Rule{
    ID:       "public-health",
    Scope:    "",
    Resource: &auth.Resource{Endpoint: "Greeter.Health"},
    Access:   auth.AccessGranted,
})

// 3. Wrap service with auth handler
service := micro.NewService(
    micro.Name("greeter"),
    micro.WrapHandler(
        authWrapper.AuthHandler(authWrapper.HandlerOptions{
            Auth:          authProvider,
            Rules:         rules,
            SkipEndpoints: []string{"Greeter.Health"},
        }),
    ),
)
```

### Client Setup

```go
// 1. Get or generate token
token := os.Getenv("TOKEN")

// 2. Wrap client with auth
service := micro.NewService(
    micro.Name("greeter.client"),
    micro.WrapClient(
        authWrapper.FromToken(token),
    ),
)

// 3. Make calls (token automatically added)
greeterClient := pb.NewGreeterService("greeter", service.Client())
rsp, err := greeterClient.Hello(ctx, &pb.Request{Name: "John"})
```

### Handler Implementation

```go
func (g *Greeter) Hello(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
    // Get account from context (added by auth wrapper)
    acc, ok := auth.AccountFromContext(ctx)
    if !ok {
        return errors.Unauthorized("greeter", "authentication required")
    }

    rsp.Msg = "Hello, " + acc.ID + "!"
    return nil
}
```

## Auth Wrapper Features

### Server Wrapper (`AuthHandler`)

- **Token Extraction**: Reads `Authorization: Bearer <token>` from metadata
- **Token Verification**: Validates token using `auth.Inspect()`
- **Authorization**: Checks permissions using `rules.Verify()`
- **Context Injection**: Adds account to context for handlers
- **Error Handling**: Returns 401/403 with clear error messages
- **Skip Endpoints**: Allows public endpoints without auth

### Client Wrapper (`AuthClient`)

- **Automatic Token Injection**: Adds Bearer token to all requests
- **Context-Aware**: Can extract account from context
- **Static Token**: Use `FromToken()` for pre-generated tokens
- **Dynamic Token**: Use `FromContext()` to generate per-request

## Auth Strategies

### 1. All Endpoints Protected

```go
micro.WrapHandler(
    authWrapper.AuthRequired(authProvider, rules),
)
```

### 2. Some Public Endpoints

```go
micro.WrapHandler(
    authWrapper.PublicEndpoints(authProvider, rules, []string{
        "Health.Check",
        "Status.Version",
    }),
)
```

### 3. Optional Auth (Extract but Don't Enforce)

```go
micro.WrapHandler(
    authWrapper.AuthOptional(authProvider),
)
```

## Authorization Rules

### Grant Public Access

```go
rules.Grant(&auth.Rule{
    ID:       "public",
    Scope:    "",  // No scope = public
    Resource: &auth.Resource{Endpoint: "Health.Check"},
    Access:   auth.AccessGranted,
})
```

### Require Authentication

```go
rules.Grant(&auth.Rule{
    ID:       "authenticated",
    Scope:    "*",  // Any authenticated user
    Resource: &auth.Resource{Endpoint: "*"},
    Access:   auth.AccessGranted,
})
```

### Require Specific Scope

```go
rules.Grant(&auth.Rule{
    ID:       "admin-only",
    Scope:    "admin",  // Only admin scope
    Resource: &auth.Resource{Endpoint: "Admin.*"},
    Access:   auth.AccessGranted,
})
```

### Deny Access

```go
rules.Grant(&auth.Rule{
    ID:       "deny-delete",
    Scope:    "*",
    Resource: &auth.Resource{Endpoint: "User.Delete"},
    Access:   auth.AccessDenied,
    Priority: 100,  // Higher priority = evaluated first
})
```

## Testing Without Server

You can test auth logic without a running server:

```go
// Create auth provider
authProvider := noop.NewAuth()

// Generate account
acc, _ := authProvider.Generate("test-user", auth.WithScopes("admin"))

// Generate token
token, _ := authProvider.Token(auth.WithCredentials(acc.ID, acc.Secret))

// Verify token
verified, _ := authProvider.Inspect(token.AccessToken)
fmt.Println(verified.ID) // "test-user"
```

## Production Considerations

### 1. Use JWT Auth (Not Noop)

```go
authProvider, err := jwt.NewAuth(
    auth.Issuer("your-company"),
    auth.Store(store), // Persistent store for keys
)
```

### 2. Load Keys from Files

```go
privateKey, _ := os.ReadFile("/etc/secrets/jwt-private.pem")
publicKey, _ := os.ReadFile("/etc/secrets/jwt-public.pem")

authProvider, err := jwt.NewAuth(
    auth.PrivateKey(string(privateKey)),
    auth.PublicKey(string(publicKey)),
)
```

### 3. Add Gateway Auth

If using HTTP gateway:

```go
// Add auth to HTTP gateway
http.Handle("/", gateway.Handler(
    gateway.WithAuth(authProvider),
))
```

### 4. Service-to-Service Auth

Services calling other services:

```go
// Service A calls Service B with its own token
client := micro.NewService(
    micro.WrapClient(
        authWrapper.FromContext(authProvider),
    ),
)
```

### 5. Token Refresh

```go
// Check if token is expiring
if time.Until(token.Expiry) < 5*time.Minute {
    token, _ = authProvider.Token(auth.WithToken(token.RefreshToken))
}
```

## Troubleshooting

### Error: "missing authorization token"

- **Cause**: Client didn't send Authorization header
- **Fix**: Wrap client with `authWrapper.FromToken(token)`

### Error: "invalid token"

- **Cause**: Token is expired or malformed
- **Fix**: Generate a new token

### Error: "access denied"

- **Cause**: Account doesn't have required permissions
- **Fix**: Check authorization rules with `rules.List()`

### Error: "token verification failed"

- **Cause**: Server can't verify token (wrong keys, expired, etc.)
- **Fix**: Ensure server and client use same auth provider

## Next Steps

- Read the [Auth Documentation](/docs/auth)
- Explore [JWT Auth](/auth/jwt)
- Try [Custom Auth Provider](/examples/auth/custom)
- See [Multi-Tenant Auth](/examples/auth/multi-tenant)

## Summary

The auth wrappers make it easy to:

1. **Protect services**: Add `WrapHandler(AuthHandler(...))`
2. **Add authentication to clients**: Add `WrapClient(FromToken(...))`
3. **Control access**: Define rules with `rules.Grant()`
4. **Access account info**: Use `auth.AccountFromContext(ctx)`

That's it! Your microservices now have enterprise-grade authentication and authorization.
