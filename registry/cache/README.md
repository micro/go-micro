# Registry Cache

Cache is a library that provides a caching layer for the go-micro [registry](https://godoc.org/github.com/micro/go-micro/registry#Registry).

If you're looking for caching in your microservices use the [selector](https://micro.mu/docs/fault-tolerance.html#caching-discovery).

## Features

- **Caching**: Caches registry lookups with configurable TTL
- **Stale Cache Fallback**: Returns stale cached data when registry is unavailable
- **Singleflight Protection**: Deduplicates concurrent requests for the same service
- **Adaptive Throttling**: Rate limits failed lookups to prevent cache penetration (new in v5)

## Interface

```go
// Cache is the registry cache interface
type Cache interface {
	// embed the registry interface
	registry.Registry
	// stop the cache watcher
	Stop()
}
```

## Usage

### Basic Usage

```go
import (
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/cache"
)

r := registry.NewRegistry()
cache := cache.New(r)

services, _ := cache.GetService("my.service")
```

### Advanced Configuration

```go
import (
	"time"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/cache"
)

r := registry.NewRegistry()

// Configure cache with custom options
cache := cache.New(r,
	cache.WithTTL(2*time.Minute),                    // Cache TTL
	cache.WithMinimumRetryInterval(10*time.Second),  // Throttle failed lookups
)

services, _ := cache.GetService("my.service")
```

## Adaptive Throttling

The cache implements rate limiting on ALL cache refresh attempts (not just errors) to prevent overwhelming the registry. This protects against multiple scenarios:

1. **Registry failures**: When etcd is down/overloaded
2. **Rolling deployments**: When all caches expire simultaneously under high QPS
3. **Cache expiration storms**: When many services expire at once

### How It Works

- **Rate limiting**: Refresh attempts are throttled per-service using `MinimumRetryInterval` (default 5s)
- **Stale cache preference**: If stale cache exists (even if expired), return it instead of calling registry
- **No cache fallback**: If no cache exists, return `ErrNotFound` and rely on gRPC retry
- **Singleflight deduplication**: Concurrent requests are still deduplicated
- **Recovery**: Throttling is reset on successful registry lookup

### Example Scenarios

#### Scenario 1: Registry Failure with Stale Cache
```go
cache := cache.New(etcdRegistry, cache.WithMinimumRetryInterval(10*time.Second))

// Initial lookup populates cache
services, _ := cache.GetService("api")  // → Calls etcd, caches result

// Cache expires after TTL
time.Sleep(2 * time.Minute)

// Etcd fails, but we have stale cache
services, err := cache.GetService("api")  // → Returns stale cache WITHOUT calling etcd
// err == nil, services contains stale data
```

#### Scenario 2: Rolling Deployment Cache Storm
```go
// Scenario: All 1000 upstream pods watch downstream service
// Downstream does rolling deployment - last pod updated
// All 1000 upstream caches expire simultaneously
// High QPS hits the system at this moment

// First request after cache expiration
services, _ := cache.GetService("downstream")  // → Calls etcd, updates lastRefreshAttempt

// Next 999 requests arrive within MinimumRetryInterval
services, _ := cache.GetService("downstream")  // → Returns stale cache, NO etcd call
// Rate limiting prevents 999 stampede requests to etcd
```

#### Scenario 3: No Cache Available
```go
// First lookup when etcd is down (no cache exists yet)
_, err := cache.GetService("new-service")  // → Calls etcd, fails, records attempt time
// err != nil

// Immediate retry (< 10s later, still no cache)
_, err = cache.GetService("new-service")  // → Throttled, returns ErrNotFound immediately
// err == ErrNotFound

// After MinimumRetryInterval
time.Sleep(10 * time.Second)
_, err = cache.GetService("new-service")  // → Allowed to retry, calls etcd again
```

This prevents cache penetration scenarios where thousands of concurrent requests hammer a failing or overloaded registry.
