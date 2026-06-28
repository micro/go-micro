# TLS Security Update - Important Information

## What Changed

Go Micro v6 verifies TLS certificates by default. This completes the v5 security
migration where verification was opt-in.

## Current Behavior (v6.x)

**Default**: TLS certificate verification is **enabled**.
- `MICRO_TLS_SECURE` was a v5 opt-in flag and is no longer used.
- For local development with untrusted self-signed certificates, opt out
  explicitly with `MICRO_TLS_INSECURE=true` or an explicit insecure TLS config.

## Production Recommendation

For production deployments:
1. Use CA-signed certificates or distribute your private CA to every host.
2. Remove old `MICRO_TLS_SECURE` settings from v5-era manifests.
3. Do not set `MICRO_TLS_INSECURE=true` in production.
4. Consider service mesh mTLS (Istio, Linkerd) if certificate lifecycle should be
   managed outside the application.

## Migration Timeline

- **v5.x**: Insecure by default, opt-in security via `MICRO_TLS_SECURE=true`.
- **v6.x current**: Secure by default; use `MICRO_TLS_INSECURE=true` only for an
  explicit development opt-out.

## Documentation

See [SECURITY_MIGRATION.md](SECURITY_MIGRATION.html) for the detailed migration
guide.

## Questions?

Open an issue on GitHub or check the documentation at https://go-micro.dev/docs/.
