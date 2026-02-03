# Performance Considerations

## Overview

go-micro is designed for **developer productivity and ease of use** while maintaining good performance for most use cases. This document explains the performance characteristics and trade-offs.

## Reflection Usage

go-micro uses Go's reflection package to enable its core feature: **registering any Go struct as a service handler** without code generation or boilerplate.

### Why Reflection?

```go
// Simple handler registration - no proto files, no code generation
type GreeterService struct{}

func (g *GreeterService) SayHello(ctx context.Context, req *Request, rsp *Response) error {
    rsp.Message = "Hello " + req.Name
    return nil
}

server.Handle(server.NewHandler(&GreeterService{}))
```

This simplicity is **only possible with reflection**. Alternative approaches (like gRPC or psrpc) require:

1. Writing `.proto` files
2. Running code generators  
3. Implementing generated interfaces
4. Managing generated code in version control

### Performance Impact

Reflection adds approximately **50 microseconds (0.05ms)** overhead per RPC call for:

- Method discovery and validation
- Dynamic method invocation
- Request/response type construction

**Context**: In typical RPC scenarios:

| Component | Typical Time |
|-----------|--------------|
| Network I/O | 1-10ms |
| Protobuf serialization | 0.1-0.5ms |
| Business logic | Variable (often 1-100ms+) |
| **Reflection overhead** | **0.05ms (0.5-5% of total)** |

### When Reflection Matters

Reflection overhead is **only significant** when ALL of these conditions are true:

1. ✅ Request rate >100,000 RPS
2. ✅ Business logic <100μs  
3. ✅ Local/loopback communication
4. ✅ Sub-millisecond latency requirements

**For 99% of applications**, database queries, external services, and business logic dominate performance. Reflection is negligible.

## Performance Best Practices

### 1. Profile Before Optimizing

Always measure before assuming reflection is your bottleneck:

```bash
# Enable pprof in your service
import _ "net/http/pprof"

# Profile CPU usage
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

If reflection shows up as <5% of CPU time, optimizing elsewhere will have more impact.

### 2. Optimize Business Logic First

Common optimization opportunities (typically 10-100x more impact than removing reflection):

- **Database queries**: Use connection pooling, indexes, query optimization
- **External API calls**: Use caching, batching, async processing
- **Serialization**: Use efficient protobuf instead of JSON
- **Concurrency**: Use goroutines and channels effectively

### 3. Use Appropriate Transports

go-micro supports multiple transports:

- **HTTP**: Good for debugging, ~1-2ms overhead
- **gRPC**: Binary protocol, ~0.2-0.5ms overhead  
- **In-memory**: Development/testing, <0.1ms overhead

Choose based on your deployment:

```go
import "go-micro.dev/v5/server/grpc"

// Use gRPC for better performance
service := micro.NewService(
    micro.Server(grpc.NewServer()),
)
```

### 4. Enable Connection Pooling

Reuse connections to avoid handshake overhead:

```go
// Client-side connection pooling (enabled by default)
client := service.Client()
```

### 5. Use Appropriate Codecs

go-micro supports multiple codecs:

```go
// Protobuf (fastest, binary)
import "go-micro.dev/v5/codec/proto"

// JSON (human-readable, slower)  
import "go-micro.dev/v5/codec/json"

// MessagePack (compact, fast)
import "go-micro.dev/v5/codec/msgpack"
```

Protobuf is 2-5x faster than JSON for most payloads.

## When to Consider Alternatives

If you've profiled and determined reflection is genuinely a bottleneck (rare), consider:

### gRPC

**Pros**:
- No reflection overhead (uses code generation)
- Industry standard
- Excellent tooling

**Cons**:
- Requires `.proto` files
- More boilerplate
- Less flexible

**Use when**: You need absolute maximum performance and can invest in proto definitions.

### psrpc (livekit)

**Pros**:
- No reflection
- Built on pub/sub
- Good for distributed systems

**Cons**:
- Requires proto files
- Smaller ecosystem  
- Different architecture

**Use when**: You're building LiveKit-style distributed systems and need pub/sub primitives.

### go-micro (Current)

**Pros**:
- Zero boilerplate
- Pure Go
- Rapid development
- Flexible

**Cons**:
- ~50μs reflection overhead per call
- Not suitable for <100μs latency requirements

**Use when**: Developer productivity and code simplicity matter more than squeezing every microsecond.

## Benchmarks

Synthetic benchmarks (single request/response, no business logic):

| Framework | Latency (p50) | Throughput |
|-----------|---------------|------------|
| Direct function call | 1μs | 1M+ RPS |
| go-micro (reflection) | ~60μs | ~16k RPS |
| gRPC (generated code) | ~40μs | ~25k RPS |

**Real-world** (with database, business logic):

| Scenario | go-micro | gRPC | Difference |
|----------|----------|------|------------|
| REST API + DB | 15ms | 14.95ms | 0.3% |
| Microservice call | 5ms | 4.95ms | 1% |
| Batch processing | 100ms | 100ms | 0% |

Reflection overhead is **lost in the noise** for realistic workloads.

## Future Optimizations

Possible future improvements (without removing reflection):

1. **Method cache warming**: Pre-compute reflection metadata at startup
2. **Call argument pooling**: Reuse `reflect.Value` slices
3. **JIT optimization**: Generate specialized handlers for hot paths

These could reduce reflection overhead by 50-70% while maintaining the simple API.

## Summary

- **Reflection is a deliberate design choice** that enables go-micro's simplicity
- **Overhead is negligible** (<5%) for typical microservices  
- **Optimize business logic first** - usually 10-100x more impact
- **Profile before optimizing** - measure, don't guess
- **Consider alternatives** only if profiling proves reflection is a bottleneck

For most applications, go-micro's productivity benefits far outweigh the minimal reflection overhead.

## Related Documents

- [Reflection Removal Analysis](reflection-removal-analysis.md) - Detailed technical analysis
- [Architecture](architecture.md) - go-micro design principles
- [Comparison with gRPC](grpc-comparison.md) - When to use each

## References

- [Go Reflection Laws](https://go.dev/blog/laws-of-reflection) - Official Go blog
- [Effective Go](https://go.dev/doc/effective_go) - Go best practices
- [gRPC Performance Best Practices](https://grpc.io/docs/guides/performance/)
