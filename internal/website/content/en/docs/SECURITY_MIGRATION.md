# TLS Security Migration Guide

## Overview

Go Micro v6 verifies TLS certificates by default. This guide is for teams
upgrading from v5, where TLS verification was disabled by default for backward
compatibility.

## Current Status (v6)

**Default Behavior**: TLS certificate verification is **enabled** by default
(`InsecureSkipVerify: false`).

**What changed from v5**: v5 allowed `MICRO_TLS_SECURE=true` to opt into
certificate verification. In v6, secure verification is the default and
`MICRO_TLS_SECURE` is no longer used.

**Development escape hatch**: for local self-signed certificates only, set
`MICRO_TLS_INSECURE=true` or provide an explicit insecure TLS config.

## Migration Path from v5

### 1. Remove the old opt-in flag

Delete any use of the v5-only environment variable:

```bash
unset MICRO_TLS_SECURE
```

No replacement is required for production: verification is already on in v6.

### 2. Use the default secure config

Most services need no TLS-specific code. If you configure TLS explicitly, use a standard `crypto/tls` config with verification enabled:

```go
import (
    "crypto/tls"
    "go-micro.dev/v6/broker"
)

// Create broker with certificate verification enabled.
b := broker.NewHttpBroker(
    broker.TLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}),
)
```

### 3. Provide a custom trust root when needed

For private CAs, provide your own TLS configuration:

```go
import (
    "crypto/tls"
    "crypto/x509"
    "go-micro.dev/v6/broker"
    "os"
)

// Load CA certificates
caCert, err := os.ReadFile("/path/to/ca-cert.pem")
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

### 4. Use insecure mode only for local development

If a development environment still uses self-signed certificates that are not in
your trust store, opt out explicitly:

```bash
export MICRO_TLS_INSECURE=true
```

or in code:

```go
broker.TLSConfig(&tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12})
```

Do not use insecure mode in production.

## Production Deployment Strategy

### Rolling Upgrade Considerations

The default changed at the v6 major-version boundary. Before rolling v6 into a
fleet that uses TLS, verify that:

1. All services present certificates trusted by their peers.
2. Private or self-signed CAs are installed consistently on every host.
3. Certificates include the DNS names or IP subject alternative names used by
   clients.
4. Any deliberate development-only insecure settings are excluded from
   production manifests.

### Recommended Approach

1. **Test in Staging** with the same certificate chain and service names used in
   production.
2. **Remove v5 flags** such as `MICRO_TLS_SECURE`; they no longer control v6.
3. **Monitor for Issues**: watch for TLS handshake failures or certificate
   validation errors.
4. **Use explicit insecure mode only in dev** when a short-lived environment
   cannot yet provide trusted certificates.

### Multi-Host/Multi-Process Considerations

**Certificate Trust**: With secure mode as the default, ensure:

1. All hosts trust the same root CAs.
2. Self-signed certificates are properly distributed if used.
3. Certificate validity periods are monitored.
4. Certificate chains are complete.

**Service Mesh Alternative**: Consider using a service mesh (Istio, Linkerd, etc.) for:
- Automatic mTLS between services
- Certificate management and rotation
- No application code changes required

## Testing Your Migration

### Verify Secure Mode is Active

```go
package main

import (
    "crypto/tls"
    "fmt"
)

func main() {
    config := &tls.Config{MinVersion: tls.VersionTLS12}
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
3. For development only: use `MICRO_TLS_INSECURE=true` or an explicit insecure TLS config

### Issue: "x509: certificate has expired"

**Cause**: Server certificate has expired

**Solution**:
1. Renew the certificate
2. Implement certificate rotation
3. Monitor certificate expiry dates

### Issue: Services can't communicate after upgrading to v6

**Cause**: Certificates that v5 accepted by default are now verified.

**Solution**:
1. Ensure all services use certificates from a trusted CA
2. Distribute CA certificates to all nodes
3. Verify certificate SANs match service addresses
4. Use insecure mode only as a temporary local-development workaround

## Questions?

For issues or questions about TLS security migration, open an issue on GitHub or
check the documentation at https://go-micro.dev/docs/.
