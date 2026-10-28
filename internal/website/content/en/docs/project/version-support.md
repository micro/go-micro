---
title: "Version Support"
---

# Version support

This page is the authoritative support policy for Go Micro releases.

| Release | Status | Changes accepted |
| --- | --- | --- |
| v6 | Active | Features, compatible fixes, and security fixes |
| v5 | Security-only | Critical and high-severity security fixes when a safe backport is practical |
| v4 and earlier | End of life | None |

## Compatibility and releases

v6 follows semantic versioning. Minor and patch releases preserve the public v6
API; removals and other intentional compatibility breaks are reserved for a new
major release. Releases are made when a coherent set of changes passes the
release gates rather than on a fixed calendar. Security releases may be issued
outside that cadence.

The supported Go version is the `go` directive in `go.mod`. CI also uses the
`toolchain` version recorded there. A release may raise the minimum Go version
in a minor release when required for security or maintained dependencies; the
release notes will call this out.

## Backports

Fixes land on the active major first. Maintainers assess security-only
backports for severity, exploitability, compatibility risk, and whether the
dependency ecosystem still supports the older branch. A security-only status
does not guarantee that every fix can be backported. End-of-life branches do
not receive fixes.

## Upgrade guidance

- [v5 to v6 migration guide](../guides/migration/v5-to-v6.md)
- [v7 proposal](../v7.md) (design review; not a release commitment)

If a supported upgrade is blocked, open an issue with the exact Go Micro and Go
versions, integrations in use, and the incompatible behavior. This information
also feeds the project's aggregate adoption review; runtime telemetry is not
required.
