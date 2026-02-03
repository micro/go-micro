# TLS Security Update - Important Information

## What Changed

The TLS configuration in go-micro now includes a security deprecation warning.

## Current Behavior (v5.x)

**Default**: TLS certificate verification is **disabled** for backward compatibility
- This maintains existing behavior to avoid breaking production deployments
- A deprecation warning is logged once per process startup

**Why**: Changing the default to secure would be a **breaking change** that could disrupt:
- Production systems during routine upgrades
- Distributed systems with mixed versions
- Services using self-signed certificates

## How to Enable Security (Recommended)

### Option 1: Environment Variable

```bash
export MICRO_TLS_SECURE=true
```

### Option 2: Use SecureConfig

```go
import (
    "go-micro.dev/v5/broker"
    mls "go-micro.dev/v5/util/tls"
)

broker := broker.NewHttpBroker(
    broker.TLSConfig(mls.SecureConfig()),
)
```

## Migration Timeline

- **v5.x (Current)**: Insecure by default, opt-in security via `MICRO_TLS_SECURE=true`
- **v6.x (Future)**: Secure by default (breaking change with major version bump)

## Why This Approach?

This addresses the concerns raised about:

1. **Major version requirements**: No breaking change in v5, deferred to v6
2. **Cross-host compatibility**: All hosts use same default behavior
3. **Production safety**: Existing deployments continue working during upgrades
4. **Migration path**: Clear opt-in path with documentation

## Documentation

See [SECURITY_MIGRATION.md](./SECURITY_MIGRATION.md) for detailed migration guide.

## Security Recommendation

For production deployments:
1. Test with `MICRO_TLS_SECURE=true` in staging
2. Use proper CA-signed certificates
3. Consider service mesh (Istio, Linkerd) for automatic mTLS
4. Plan migration before v6 release

## Questions?

Open an issue on GitHub or check the documentation at https://go-micro.dev/docs/
