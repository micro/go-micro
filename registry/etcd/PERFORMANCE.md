# Etcd Registry Performance Improvements

This document describes the improvements made to address etcd authentication performance issues and cache penetration problems.

## Problem Statement

### Background
When etcd server authentication is enabled, a serious performance bottleneck can occur at scale. This was observed in production environments with 4000+ service pods.

### Issues Identified

#### 1. High Authentication QPS
- **Root Cause**: The etcd registry used `KeepAliveOnce` for lease renewal, which requires a new authentication request for each call
- **Impact**: With 4000+ pods registering every 30s (default RegisterInterval), this creates ~110 QPS of authentication requests
- **Limitation**: A typical 3-node etcd cluster (64C 256G HDD) can only handle ~100 QPS for authentication
- **Result**: Authentication requests overwhelm etcd, causing KeepAlive failures and service deregistrations

#### 2. Cache Penetration
- **Trigger**: When KeepAlive fails, services deregister from etcd
- **Chain Reaction**: 
  1. Registry watcher detects deletions
  2. Cache is cleared based on delete events
  3. All subsequent service lookups hit etcd directly (cache miss)
  4. Etcd is already overloaded, causing more failures
- **Result**: Cascading failure where all gRPC requests fail

## Solution

### 1. Use Long-Lived KeepAlive Channels

**Change**: Replaced `KeepAliveOnce` with `KeepAlive`

**Implementation**:
- Added keepalive channel management to `etcdRegistry` struct
- Created `startKeepAlive()` method that establishes a long-lived keepalive stream
- Modified `registerNode()` to reuse existing keepalive channels
- Added `stopKeepAlive()` for proper cleanup on deregistration

**Benefits**:
- **97% reduction in auth requests**: From ~110 QPS to ~3-4 QPS (4000 pods / TTL period)
- **Single authentication per lease**: KeepAlive authenticates once when establishing the stream
- **Automatic renewal**: Etcd sends keepalive responses automatically through the channel

**Code Changes**:
```go
// Before: New auth request every heartbeat
if _, err := e.client.KeepAliveOnce(context.TODO(), leaseID); err != nil {
    // handle error
}

// After: Single auth request, reused channel
if err := e.startKeepAlive(s.Name+node.Id, leaseID); err != nil {
    // handle error
}
```

### 2. Verify Cache Penetration Protection

**Existing Protection**: The registry cache already uses `singleflight` pattern to prevent stampede

**How it Works**:
- When cache expires/is empty, first request triggers etcd query
- Concurrent requests for same service wait for the first request to complete
- All waiting requests receive the same result
- Only ONE etcd query happens regardless of concurrent request count

**Additional Safety**:
- Stale cache is returned when etcd fails (if cache data exists)
- Prevents cascading failures by avoiding repeated failed requests to etcd

**Verification**:
Added comprehensive tests to confirm this behavior works correctly under load.

## Performance Impact

### Authentication Load Reduction
- **Before**: 4000 pods × (1 auth / 30s) = ~133 auth/sec
- **After**: 4000 pods × (1 auth / lease_ttl) ≈ 3-4 auth/sec (assuming 15min lease TTL)
- **Reduction**: ~97%

### Cache Penetration Prevention
- **Before**: When cache clears, 1000s of concurrent requests → 1000s of etcd queries
- **After**: When cache clears, 1000s of concurrent requests → 1 etcd query (singleflight)
- **Reduction**: ~99.9%

## Testing

### Unit Tests
1. **TestKeepAliveManagement**: Validates keepalive lifecycle
   - Verifies channels are created on registration
   - Confirms channels are cleaned up on deregistration
   
2. **TestKeepAliveReducesAuthRequests**: Confirms channel reuse
   - Multiple re-registrations use the same keepalive channel
   - Validates auth request reduction

3. **TestKeepAliveChannelReconnection**: Tests error handling
   - Verifies proper cleanup when keepalive channel closes

4. **TestSingleflightPreventsStampede**: Validates cache behavior
   - 10 concurrent requests → 1 etcd query
   
5. **TestStaleCacheOnError**: Confirms graceful degradation
   - Returns stale cache when etcd fails

6. **TestCachePenetrationPrevention**: End-to-end validation
   - 50 concurrent requests during etcd failure → 1 etcd query
   - All requests receive stale cache

### Integration Tests
- CI workflow runs tests against real etcd instance
- Validates behavior with actual etcd keepalive channels
- Tests run with race detector enabled

## Migration Guide

### For Library Users
No code changes required! The improvements are transparent:
- Existing applications automatically benefit from reduced auth load
- No API changes to `registry.Registry` interface

### For Plugin Developers
If you maintain a custom registry plugin:
- Consider implementing long-lived keepalive channels
- Ensure your cache implementation uses singleflight pattern
- Add tests for concurrent access patterns

## Monitoring Recommendations

### Key Metrics to Track
1. **Etcd Authentication Rate**: Should drop by ~97%
2. **Etcd Query Rate**: Monitor for stampede prevention
3. **Service Registration Success Rate**: Should improve under load
4. **Cache Hit Rate**: Should remain high even during etcd issues

### Expected Behavior
- **Normal Operation**: Low auth QPS, high cache hit rate
- **During Etcd Issues**: Stale cache served, limited etcd queries
- **After Recovery**: Cache refreshes gradually, no stampede

## Related Issues

- Original Issue: [BUG] etcd authentication performance issue and registry cache penetration
- Etcd Documentation: https://etcd.io/docs/latest/learning/api/#lease-keepalive
- Singleflight Pattern: https://pkg.go.dev/golang.org/x/sync/singleflight

## Security Considerations

- **No Authentication Bypass**: Changes only reduce frequency, not security
- **Proper Cleanup**: Keepalive channels properly closed on deregistration
- **Race Condition Free**: All map operations properly synchronized
- **No Resource Leaks**: Goroutines terminate when channels close

## Future Enhancements

Potential improvements for consideration:
1. **Adaptive TTL**: Adjust keepalive frequency based on load
2. **Circuit Breaker**: Temporarily stop queries when etcd is degraded
3. **Metrics**: Expose keepalive channel count, auth rate, etc.
4. **Backoff**: Exponential backoff on keepalive failures
