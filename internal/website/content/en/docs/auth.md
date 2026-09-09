---
title: "Authentication"
description: "Authentication and authorization in Go Micro"
---

Go Micro provides a pluggable authentication interface that covers JWT tokens, account management, scoped access control, and custom auth providers.

## Overview

The `auth` package in `go-micro.dev/v6/auth` defines the core interfaces:

- **`Auth`** — the main authentication interface
- **`Account`** — represents an authenticated entity with scopes
- **`Token`** — a JWT-based access token

## Quick Start

```go
import "go-micro.dev/v6/auth"

// Use the default auth implementation
a := auth.DefaultAuth

// Generate a token
tok, err := a.Generate("admin")
if err != nil {
    // handle error
}

// Validate a token
account, err := a.Inspect(tok.Token)
if err != nil {
    // handle error
}
```

## Next Steps

- [JWT Auth](/auth/jwt)
- [Custom Auth Provider](/examples/auth/custom)
- [CLI Gateway](/docs/guides/cli-gateway.html)
