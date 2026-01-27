---
layout: default
---

# Health Checks

The `health` package provides health check functionality for microservices, including Kubernetes-style liveness and readiness probes.

## Quick Start

```go
import "go-micro.dev/v5/health"

func main() {
    // Register health checks
    health.Register("database", health.PingCheck(db.Ping))
    health.Register("cache", health.TCPCheck("localhost:6379", time.Second))
    
    // Add health endpoints
    mux := http.NewServeMux()
    health.RegisterHandlers(mux)  // Registers /health, /health/live, /health/ready
    
    http.ListenAndServe(":8080", mux)
}
```

## Endpoints

| Endpoint | Purpose | Returns 200 when |
|----------|---------|------------------|
| `/health` | Overall health status | All critical checks pass |
| `/health/live` | Kubernetes liveness probe | Service is running |
| `/health/ready` | Kubernetes readiness probe | All critical checks pass |

## Response Format

```json
{
  "status": "up",
  "checks": [
    {
      "name": "database",
      "status": "up",
      "duration": 1234567
    },
    {
      "name": "cache",
      "status": "up", 
      "duration": 567890
    }
  ],
  "info": {
    "go_version": "go1.22.0",
    "go_os": "linux",
    "go_arch": "amd64",
    "version": "1.0.0"
  }
}
```

When unhealthy:
- HTTP status: 503 Service Unavailable
- `status`: `"down"`
- Failed checks include an `error` field

## Built-in Checks

### PingCheck

For database connections with a `Ping()` method:

```go
health.Register("postgres", health.PingCheck(db.Ping))
health.Register("mysql", health.PingContextCheck(db.PingContext))
```

### TCPCheck

Verify TCP connectivity:

```go
health.Register("redis", health.TCPCheck("localhost:6379", time.Second))
health.Register("kafka", health.TCPCheck("kafka:9092", 2*time.Second))
```

### HTTPCheck

Verify an HTTP endpoint returns 200:

```go
health.Register("api", health.HTTPCheck("http://api.internal/health", time.Second))
```

### DNSCheck

Verify DNS resolution:

```go
health.Register("dns", health.DNSCheck("api.example.com"))
```

### CustomCheck

Any function returning an error:

```go
health.Register("disk", health.CustomCheck(func() error {
    var stat syscall.Statfs_t
    if err := syscall.Statfs("/", &stat); err != nil {
        return err
    }
    freeGB := stat.Bavail * uint64(stat.Bsize) / 1e9
    if freeGB < 1 {
        return fmt.Errorf("low disk space: %dGB free", freeGB)
    }
    return nil
}))
```

## Critical vs Non-Critical Checks

By default, all checks are critical. A critical check failure marks the service as not ready.

For non-critical checks (monitoring only):

```go
health.RegisterCheck(health.Check{
    Name:     "external-api",
    Check:    health.HTTPCheck("https://api.external.com/status", 5*time.Second),
    Critical: false,  // Won't affect readiness
    Timeout:  5 * time.Second,
})
```

## Timeouts

Default timeout is 5 seconds. Override per-check:

```go
health.RegisterCheck(health.Check{
    Name:    "slow-db",
    Check:   health.PingCheck(db.Ping),
    Timeout: 10 * time.Second,
})
```

## Adding Service Info

Include metadata in health responses:

```go
health.SetInfo("version", "1.0.0")
health.SetInfo("commit", "abc123")
health.SetInfo("service", "users")
```

## Kubernetes Configuration

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: app
    livenessProbe:
      httpGet:
        path: /health/live
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /health/ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
```

## Integration with micro run

When using `micro run` with a `micro.mu` config that specifies ports, the runner waits for `/health` to return 200 before starting dependent services:

```
service database
    path ./database
    port 8081

service api
    path ./api
    port 8080
    depends database
```

The `api` service won't start until `database`'s `/health` endpoint is ready.

## Programmatic Usage

```go
// Check readiness in code
if health.IsReady(ctx) {
    // Service is healthy
}

// Get full health status
resp := health.Run(ctx)
fmt.Printf("Status: %s\n", resp.Status)
for _, check := range resp.Checks {
    fmt.Printf("  %s: %s (%v)\n", check.Name, check.Status, check.Duration)
}
```

## Best Practices

1. **Keep checks fast** - Health endpoints are called frequently
2. **Use timeouts** - Don't let slow dependencies block health checks
3. **Non-critical for optional deps** - External APIs, caches that have fallbacks
4. **Critical for required deps** - Databases, message queues
5. **Include version info** - Helps debugging in production
