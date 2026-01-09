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

When the registry is unavailable and there's no stale cache to return, the cache now throttles retry attempts to prevent overwhelming the registry:

- **Default behavior**: Failed lookups are throttled for 5 seconds
- **Configurable**: Use `WithMinimumRetryInterval()` to adjust the interval
- **Protection scope**: Per-service throttling (different services can be retried independently)
- **Recovery**: Throttling is cleared on successful lookup

### Example Scenario

```go
cache := cache.New(etcdRegistry, cache.WithMinimumRetryInterval(10*time.Second))

// First lookup when etcd is down
_, err := cache.GetService("api")  // → Calls etcd, fails, records failure time

// Immediate retry (< 10s later)
_, err = cache.GetService("api")  // → Throttled, returns ErrNotFound immediately

// After 10 seconds
time.Sleep(10 * time.Second)
_, err = cache.GetService("api")  // → Allowed to retry, calls etcd again
```

This prevents cache penetration scenarios where thousands of concurrent requests hammer a failing registry.
