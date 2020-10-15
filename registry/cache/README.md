# Registry Cache 

Cache is a library that provides a caching layer for the go-micro [registry](https://godoc.org/github.com/micro/go-micro/registry#Registry).

If you're looking for caching in your microservices use the [selector](https://micro.mu/docs/fault-tolerance.html#caching-discovery).

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

```go
import "github.com/micro/go-micro/registry/cache"

# create a new cache
c := cache.New(registry)

# get a service from the cache
services, _ := c.GetService("helloworld")
```
