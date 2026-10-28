# Security Policy

## Supported Versions

We actively support the following versions of go-micro:

| Version | Support level |
| ------- | ------------- |
| 6.x     | Active development and security fixes |
| 5.x     | Security fixes only |
| 4.x and earlier | End of life |

The complete policy, including Go toolchain support, release cadence, backports,
and migration guidance, is documented in [Version Support](internal/website/content/en/docs/project/version-support.md).

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

### How to Report

Use GitHub's private security advisory feature:
https://github.com/micro/go-micro/security/advisories/new

This keeps vulnerability reports private, ties follow-up to the affected repository, and avoids relying on project email routing.

### What to Include

Please include as much of the following information as possible:

- Type of vulnerability (e.g., RCE, XSS, SQL injection, etc.)
- Full paths of source file(s) related to the vulnerability
- Location of the affected source code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

### Response Timeline

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 5 business days
- **Fix Timeline**: Depends on severity
  - Critical: 7 days
  - High: 14 days
  - Medium: 30 days
  - Low: Next release cycle

### Disclosure Policy

- We follow **coordinated disclosure**
- We'll work with you to understand and fix the issue
- We'll credit you in the security advisory (unless you prefer to remain anonymous)
- Please give us reasonable time to fix before public disclosure
- We'll publish a security advisory on GitHub when the fix is released

## Security Best Practices

When using go-micro in production:

### TLS/Transport Security

```go
import "go-micro.dev/v6/transport"

// Enable TLS verification (recommended)
os.Setenv("MICRO_TLS_SECURE", "true")

// Or use SecureConfig explicitly
tlsConfig := transport.SecureConfig()
```

See [TLS Security Update](internal/website/docs/TLS_SECURITY_UPDATE.md) for details.

### Authentication

```go
import "go-micro.dev/v6/auth"

// Use JWT authentication
service := micro.NewService(
    micro.Auth(auth.NewAuth()),
)
```

### Input Validation

Always validate and sanitize inputs in your handlers:

```go
func (h *Handler) Create(ctx context.Context, req *Request, rsp *Response) error {
    // Validate input
    if req.Name == "" {
        return errors.BadRequest("handler.create", "name is required")
    }
    
    // Sanitize and process
    // ...
}
```

### Rate Limiting

Implement rate limiting for public-facing services:

```go
import "go-micro.dev/v6/client"

// Client-side rate limiting
client.NewClient(
    client.RequestTimeout(time.Second * 5),
    client.Retries(3),
)
```

### Secrets Management

Never commit secrets to version control:

```go
// Good: Use environment variables
apiKey := os.Getenv("API_KEY")

// Better: Use a secrets manager
import "github.com/hashicorp/vault/api"
```

### Dependency Security

Regularly update dependencies:

```bash
# Check for vulnerabilities
go list -json -m all | nancy sleuth

# Update dependencies
go get -u ./...
go mod tidy
```

## Known Security Considerations

### Reflection Usage

go-micro uses reflection for automatic handler registration. While this is a deliberate design choice for developer productivity, be aware:

- Type safety is enforced at runtime, not compile time
- Malformed requests won't crash services (errors are returned)
- See [Performance Considerations](internal/website/docs/performance.md)

### TLS Certificate Verification

**Default behavior in v6**: TLS certificate verification is enabled. Do not
disable verification in production. Older releases may require secure mode to
be enabled explicitly:

```bash
export MICRO_TLS_SECURE=true
```

See the v5 to v6 migration guide before upgrading security-sensitive services.

## Security Updates

Security updates are published as:
- GitHub Security Advisories
- Release notes with `[SECURITY]` prefix
- CVE entries for critical issues

Subscribe to releases: https://github.com/micro/go-micro/releases

## Bug Bounty

We currently do not offer a bug bounty program, but we greatly appreciate responsible disclosure and will publicly credit researchers who report valid security issues.

## Questions?

For security questions that are not vulnerabilities, please:
- Open a discussion: https://github.com/micro/go-micro/discussions
- Join Discord: https://discord.gg/G8Gk5j3uXr

