---
name: Performance issue
about: Report a performance problem or regression
title: '[PERFORMANCE] '
labels: performance
assignees: ''
---

## Performance Issue

**Symptom:**
Describe the performance problem (e.g., high latency, memory leak, CPU usage)

**Expected Performance:**
What performance did you expect?

## Benchmarks

Please provide benchmarks or profiling data:

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.

# Memory profiling
go test -memprofile=mem.prof -bench=.

# Results
```

**Before/After comparison (if applicable):**
- Before: X req/sec, Y ms latency
- After: X req/sec, Y ms latency

## Code Sample

```go
// Minimal code that demonstrates the performance issue
```

## Environment
- Go Micro version: [e.g. v5.3.0]
- Go version: [run `go version`]
- Hardware: [e.g. 4 CPU, 8GB RAM]
- OS: [e.g. Ubuntu 22.04]
- Load: [e.g. 1000 req/sec, 100 concurrent connections]

## Profiling Data

Attach pprof profiles if available:
- CPU profile
- Memory profile
- Goroutine dump

## Additional Context

Add any other context about the performance issue.

## Resources
- [Performance Guide](https://github.com/micro/go-micro/tree/master/internal/website/docs/performance.md)
- [Benchmarking](https://pkg.go.dev/testing#hdr-Benchmarks)
