# Auth Wrapper

The auth wrapper package provides server and client wrappers for adding authentication and authorization to your go-micro services.

## Installation

```go
import "go-micro.dev/v5/wrapper/auth"
```

## Overview

The auth wrapper consists of three main components:

1. **Server Wrapper** (`AuthHandler`) - Protects service endpoints
2. **Client Wrapper** (`AuthClient`) - Adds auth tokens to requests
3. **Metadata Helpers** - Extract/inject tokens from/to metadata

## Server Wrapper

The server wrapper enforces authentication and authorization on incoming requests.

### Basic Usage

```go
import (
    "go-micro.dev/v5"
    "go-micro.dev/v5/auth/jwt"
    authWrapper "go-micro.dev/v5/wrapper/auth"
)

func main() {
    // Create auth provider
    authProvider, _ := jwt.NewAuth()

    // Create authorization rules
    rules := auth.NewRules()

    // Wrap service with auth
    service := micro.NewService(
        micro.Name("myservice"),
        micro.WrapHandler(
            authWrapper.AuthHandler(authWrapper.HandlerOptions{
                Auth:  authProvider,
                Rules: rules,
            }),
        ),
    )

    service.Run()
}
```

### Configuration Options

```go
type HandlerOptions struct {
    // Auth provider for token verification (required)
    Auth auth.Auth

    // Rules for authorization checks (optional)
    Rules auth.Rules

    // SkipEndpoints is a list of endpoints that don't require auth
    // Format: "Service.Method" e.g., "Greeter.Hello"
    SkipEndpoints []string
}
```

### Auth Flow

For each incoming request:

1. **Check Skip List**: If endpoint in `SkipEndpoints`, skip auth
2. **Extract Token**: Get `Authorization: Bearer <token>` from metadata
3. **Verify Token**: Call `auth.Inspect(token)` to get account
4. **Check Authorization**: Call `rules.Verify(account, resource)`
5. **Inject Context**: Add account to context with `auth.ContextWithAccount()`
6. **Call Handler**: Proceed to actual handler

**Errors:**
- `401 Unauthorized` - Missing or invalid token
- `403 Forbidden` - Token valid but insufficient permissions

### Helper Functions

#### AuthRequired

Enforce auth on all endpoints (no public endpoints):

```go
micro.WrapHandler(
    authWrapper.AuthRequired(authProvider, rules),
)
```

#### PublicEndpoints

Allow specific endpoints to be public:

```go
micro.WrapHandler(
    authWrapper.PublicEndpoints(authProvider, rules, []string{
        "Health.Check",
        "Status.Version",
    }),
)
```

#### AuthOptional

Extract auth if present but don't enforce (useful for endpoints that behave differently for authenticated users):

```go
micro.WrapHandler(
    authWrapper.AuthOptional(authProvider),
)
```

With `AuthOptional`, the handler can check:

```go
func (s *Service) Hello(ctx context.Context, req *Request, rsp *Response) error {
    if acc, ok := auth.AccountFromContext(ctx); ok {
        rsp.Msg = "Hello, " + acc.ID
    } else {
        rsp.Msg = "Hello, anonymous"
    }
    return nil
}
```

## Client Wrapper

The client wrapper adds authentication tokens to outgoing requests.

### Basic Usage

```go
import (
    "go-micro.dev/v5"
    "go-micro.dev/v5/client"
    authWrapper "go-micro.dev/v5/wrapper/auth"
)

func main() {
    token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

    service := micro.NewService(
        micro.Name("myclient"),
        micro.WrapClient(
            authWrapper.FromToken(token),
        ),
    )

    service.Init()

    // All calls now include the token
    client := pb.NewMyServiceClient("myservice", service.Client())
    rsp, err := client.SomeMethod(ctx, &pb.Request{})
}
```

### Configuration Options

```go
type ClientOptions struct {
    // Auth provider for token generation (optional)
    Auth auth.Auth

    // Static token to use (optional)
    // If not provided, will try to extract from context
    Token string
}
```

### Helper Functions

#### FromToken

Use a static token for all requests:

```go
client.Wrap(
    authWrapper.FromToken("eyJhbGciOi..."),
)
```

Best for:
- Pre-generated tokens
- Service accounts
- Long-lived tokens

#### FromContext

Extract account from context and generate token per-request:

```go
client.Wrap(
    authWrapper.FromContext(authProvider),
)
```

Best for:
- Service-to-service auth
- Dynamic token generation
- Request context propagation

Example:

```go
func (s *Service) HandleRequest(ctx context.Context, req *Request, rsp *Response) error {
    // Account already in context from incoming request

    // Client wrapper extracts account and generates token
    client := pb.NewOtherService("other", s.Client())

    // Token automatically added
    otherRsp, err := client.SomeMethod(ctx, &pb.OtherRequest{})

    return nil
}
```

## Metadata Helpers

Low-level helpers for working with auth tokens in metadata.

### TokenFromMetadata

Extract Bearer token from request metadata:

```go
import (
    "go-micro.dev/v5/metadata"
    authWrapper "go-micro.dev/v5/wrapper/auth"
)

func handler(ctx context.Context, req *Request, rsp *Response) error {
    md, _ := metadata.FromContext(ctx)

    token, err := authWrapper.TokenFromMetadata(md)
    if err != nil {
        return err // ErrMissingToken or ErrInvalidToken
    }

    // Use token...
}
```

**Returns:**
- Token string (without "Bearer " prefix)
- `ErrMissingToken` - No Authorization header found
- `ErrInvalidToken` - Not in "Bearer <token>" format

### TokenToMetadata

Add Bearer token to outgoing request metadata:

```go
md := metadata.Metadata{}
md = authWrapper.TokenToMetadata(md, "eyJhbGciOi...")

ctx := metadata.NewContext(context.Background(), md)

// Make RPC call with metadata
client.Call(ctx, req, rsp)
```

### AccountFromMetadata

Extract token and verify in one step:

```go
func handler(ctx context.Context, req *Request, rsp *Response) error {
    md, _ := metadata.FromContext(ctx)

    account, err := authWrapper.AccountFromMetadata(md, authProvider)
    if err != nil {
        return errors.Unauthorized("myservice", "invalid auth")
    }

    // Use account...
    log.Printf("Request from: %s", account.ID)
}
```

This combines:
1. `TokenFromMetadata(md)`
2. `authProvider.Inspect(token)`

## Complete Example

### Server

```go
package main

import (
    "context"
    "go-micro.dev/v5"
    "go-micro.dev/v5/auth"
    "go-micro.dev/v5/auth/jwt"
    authWrapper "go-micro.dev/v5/wrapper/auth"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *Request, rsp *Response) error {
    // Get authenticated account
    acc, ok := auth.AccountFromContext(ctx)
    if !ok {
        return errors.Unauthorized("greeter", "auth required")
    }

    rsp.Msg = "Hello, " + acc.ID
    return nil
}

func main() {
    authProvider, _ := jwt.NewAuth()
    rules := auth.NewRules()

    service := micro.NewService(
        micro.Name("greeter"),
        micro.WrapHandler(
            authWrapper.PublicEndpoints(authProvider, rules, []string{
                "Greeter.Health",
            }),
        ),
    )

    pb.RegisterGreeterHandler(service.Server(), &Greeter{})
    service.Run()
}
```

### Client

```go
package main

import (
    "context"
    "go-micro.dev/v5"
    authWrapper "go-micro.dev/v5/wrapper/auth"
)

func main() {
    token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

    service := micro.NewService(
        micro.WrapClient(
            authWrapper.FromToken(token),
        ),
    )

    client := pb.NewGreeterService("greeter", service.Client())
    rsp, _ := client.Hello(context.Background(), &pb.Request{})
}
```

## Testing

### Mock Auth for Tests

```go
import "go-micro.dev/v5/auth/noop"

func TestService(t *testing.T) {
    // Use noop auth for testing (always grants access)
    authProvider := noop.NewAuth()

    service := micro.NewService(
        micro.WrapHandler(
            authWrapper.AuthHandler(authWrapper.HandlerOptions{
                Auth: authProvider,
            }),
        ),
    )

    // Test your service...
}
```

### Generate Test Tokens

```go
func TestWithAuth(t *testing.T) {
    authProvider := noop.NewAuth()

    // Generate test account
    acc, _ := authProvider.Generate("test-user")

    // Generate token
    token, _ := authProvider.Token(
        auth.WithCredentials(acc.ID, acc.Secret),
    )

    // Use token in tests
    client := micro.NewService(
        micro.WrapClient(
            authWrapper.FromToken(token.AccessToken),
        ),
    )
}
```

## Integration with Gateway

If you're using the HTTP gateway (`micro server`), auth is automatically integrated:

```bash
# Gateway enforces auth on HTTP requests
micro server --auth jwt
```

The gateway:
1. Extracts Bearer token from HTTP `Authorization` header
2. Verifies token
3. Adds account to metadata
4. Forwards to service (service still checks with wrapper)

## Best Practices

### 1. Always Use Server Wrapper

Even if using gateway auth, still wrap your services:

```go
// ✅ Good: Defense in depth
micro.WrapHandler(authWrapper.AuthHandler(...))

// ❌ Bad: Only rely on gateway
// (services can be called directly, bypassing gateway)
```

### 2. Use Strong Auth in Production

```go
// ✅ Production
authProvider, _ := jwt.NewAuth(
    auth.Issuer("your-company"),
    auth.PrivateKey(privateKey),
    auth.PublicKey(publicKey),
)

// ❌ Development only
authProvider := noop.NewAuth()
```

### 3. Scope Your Rules

```go
// ✅ Good: Specific scopes
rules.Grant(&auth.Rule{
    Scope:    "admin",
    Resource: &auth.Resource{Endpoint: "Admin.*"},
})

// ⚠️ Risky: Too broad
rules.Grant(&auth.Rule{
    Scope:    "*",
    Resource: &auth.Resource{Endpoint: "*"},
})
```

### 4. Check Account in Handlers

```go
// ✅ Good: Verify account exists
func (s *Service) Delete(ctx context.Context, req *Request, rsp *Response) error {
    acc, ok := auth.AccountFromContext(ctx)
    if !ok || acc.ID != req.UserID {
        return errors.Forbidden("service", "can only delete own data")
    }
    // ...
}
```

### 5. Use AuthOptional for Mixed Endpoints

```go
// ✅ Good: Works for both auth and no-auth
func (s *Service) GetProfile(ctx context.Context, req *Request, rsp *Response) error {
    if acc, ok := auth.AccountFromContext(ctx); ok {
        // Authenticated: return private profile
        rsp.Profile = s.getPrivateProfile(acc.ID)
    } else {
        // Public: return limited profile
        rsp.Profile = s.getPublicProfile(req.UserID)
    }
    return nil
}
```

## Troubleshooting

### Issue: Handler receives requests without auth

**Check:**
1. Is wrapper applied? `micro.WrapHandler(authWrapper.AuthHandler(...))`
2. Is endpoint in skip list? Check `SkipEndpoints`
3. Is service registered correctly?

### Issue: Client gets 401 errors

**Check:**
1. Is token valid? Verify with `authProvider.Inspect(token)`
2. Is client wrapper applied? `micro.WrapClient(authWrapper.FromToken(...))`
3. Is token expired? Check `token.Expiry`

### Issue: Token extraction fails

**Check:**
1. Is Authorization header present? `md.Get("Authorization")`
2. Is format correct? Must be `Bearer <token>`
3. Is metadata propagated? Check context

## API Reference

### Server Wrapper

- `AuthHandler(opts HandlerOptions) server.HandlerWrapper`
- `PublicEndpoints(auth, rules, endpoints) HandlerOptions`
- `AuthRequired(auth, rules) HandlerOptions`
- `AuthOptional(auth) server.HandlerWrapper`

### Client Wrapper

- `AuthClient(opts ClientOptions) client.Wrapper`
- `FromToken(token) client.Wrapper`
- `FromContext(auth) client.Wrapper`

### Metadata Helpers

- `TokenFromMetadata(md) (string, error)`
- `TokenToMetadata(md, token) Metadata`
- `AccountFromMetadata(md, auth) (*Account, error)`

### Constants

- `MetadataKeyAuthorization` = `"Authorization"`
- `BearerPrefix` = `"Bearer "`

### Errors

- `ErrMissingToken` - No authorization token in metadata
- `ErrInvalidToken` - Token format invalid (not "Bearer <token>")

## See Also

- [Auth Package Documentation](/auth)
- [JWT Auth Provider](/auth/jwt)
- [Authorization Rules](/auth#rules)
- [Example Usage](/examples/auth)

## License

Apache 2.0
