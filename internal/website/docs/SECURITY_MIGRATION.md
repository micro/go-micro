# TLS Security Migration Guide

## Overview

This document provides guidance for migrating to secure TLS certificate verification in go-micro v5.

## Current Status (v5)

**Default Behavior**: TLS certificate verification is **disabled** by default (`InsecureSkipVerify: true`)

**Reason**: Backward compatibility with existing deployments to avoid breaking production systems during routine upgrades.

**Security Risk**: The default behavior is vulnerable to man-in-the-middle (MITM) attacks.

## Migration Path

### Option 1: Enable Secure Mode (RECOMMENDED)

Set the environment variable to enable certificate verification:

```bash
export MICRO_TLS_SECURE=true
```

This enables proper TLS certificate verification while maintaining compatibility with v5.

### Option 2: Use SecureConfig Directly

In your code, explicitly use the secure configuration:

```go
import (
    "go-micro.dev/v5/broker"
    mls "go-micro.dev/v5/util/tls"
)

// Create broker with secure TLS config
b := broker.NewHttpBroker(
    broker.TLSConfig(mls.SecureConfig()),
)
```

### Option 3: Provide Custom TLS Configuration

For fine-grained control, provide your own TLS configuration:

```go
import (
    "crypto/tls"
    "crypto/x509"
    "go-micro.dev/v5/broker"
    "io/ioutil"
)

// Load CA certificates
caCert, err := ioutil.ReadFile("/path/to/ca-cert.pem")
if err != nil {
    log.Fatal(err)
}

caCertPool := x509.NewCertPool()
caCertPool.AppendCertsFromPEM(caCert)

// Create custom TLS config
tlsConfig := &tls.Config{
    RootCAs:    caCertPool,
    MinVersion: tls.VersionTLS12,
}

// Create broker with custom config
b := broker.NewHttpBroker(
    broker.TLSConfig(tlsConfig),
)
```

## Production Deployment Strategy

### Rolling Upgrade Considerations

The current implementation maintains backward compatibility, allowing safe rolling upgrades:

1. **Mixed Version Deployments**: v5 instances can communicate regardless of TLS security settings
2. **No Immediate Breaking Changes**: Systems continue working with existing behavior
3. **Gradual Migration**: Enable security incrementally across your infrastructure

### Recommended Approach

1. **Test in Staging**:
   ```bash
   # In staging environment
   export MICRO_TLS_SECURE=true
   ```
   
2. **Deploy with Feature Flag**: Use environment-based configuration for gradual rollout

3. **Monitor for Issues**: Watch for TLS handshake failures or certificate validation errors

4. **Full Production Rollout**: Once validated, enable across all services

### Multi-Host/Multi-Process Considerations

**Certificate Trust**: When enabling secure mode, ensure:

1. All hosts trust the same root CAs
2. Self-signed certificates are properly distributed if used
3. Certificate validity periods are monitored
4. Certificate chains are complete

**Service Mesh Alternative**: Consider using a service mesh (Istio, Linkerd, etc.) for:
- Automatic mTLS between services
- Certificate management and rotation
- No application code changes required

## Future Changes (v6)

In go-micro v6, the default will change to **secure by default**:

- `InsecureSkipVerify: false` (certificate verification enabled)
- Breaking change requiring major version bump
- Migration completed before v6 release avoids disruption

## Testing Your Migration

### Verify Secure Mode is Active

```go
package main

import (
    "fmt"
    mls "go-micro.dev/v5/util/tls"
    "os"
)

func main() {
    os.Setenv("MICRO_TLS_SECURE", "true")
    config := mls.Config()
    fmt.Printf("InsecureSkipVerify: %v (should be false)\n", config.InsecureSkipVerify)
}
```

### Test Certificate Validation

Create a test service and verify it:
- Accepts valid certificates
- Rejects invalid/self-signed certificates (when not in CA)
- Properly validates certificate chains

## Common Issues and Solutions

### Issue: "x509: certificate signed by unknown authority"

**Cause**: The server certificate is not signed by a trusted CA

**Solution**:
1. Add the CA certificate to the trusted root CAs
2. Use a properly signed certificate
3. For development only: Use `InsecureConfig()` explicitly

### Issue: "x509: certificate has expired"

**Cause**: Server certificate has expired

**Solution**:
1. Renew the certificate
2. Implement certificate rotation
3. Monitor certificate expiry dates

### Issue: Services can't communicate after enabling secure mode

**Cause**: Mixed certificate authorities or missing certificates

**Solution**:
1. Ensure all services use certificates from the same CA
2. Distribute CA certificates to all nodes
3. Verify certificate SANs match service addresses

## Questions?

For issues or questions about TLS security migration, please:
- Open an issue on GitHub
- Check the documentation at https://go-micro.dev/docs/
- Review the security guidelines

## Security Resources

- [OWASP TLS Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Transport_Layer_Protection_Cheat_Sheet.html)
- [Go TLS Documentation](https://pkg.go.dev/crypto/tls)
- [Certificate Best Practices](https://www.ssl.com/guide/ssl-best-practices/)
