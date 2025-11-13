---
layout: default
---

# Contributing

This is a rendered copy of the repository `CONTRIBUTING.md` for convenient access via the documentation site.

## Overview

Go Micro welcomes contributions of all kinds: code, documentation, examples, and plugins.

## Quick Start

```bash
git clone https://github.com/micro/go-micro.git
cd go-micro
go mod download
go test ./...
```

## Process

1. Fork and create a feature branch
2. Make focused changes with tests
3. Run linting and full test suite
4. Open a PR describing motivation and approach

## Commit Format

Use conventional commits:

```
feat(registry): add consul health check
fix(broker): prevent reconnect storm
```

## Testing

Run unit tests:
```bash
go test ./...
```
Run race/coverage:
```bash
go test -race -coverprofile=coverage.out ./...
```

## Plugins

Place new plugins under the appropriate interface directory (e.g. `registry/consul/`). Include tests and usage examples. Document env vars and options.

## Documentation

Docs live in `internal/website/docs/`. Add new examples under `internal/website/docs/examples/`.

## Help & Questions

Use GitHub Discussions or the issue templates. For general usage questions open a "Question" issue.

## Full Guide

For complete details see the repository copy of the guide on GitHub.

- View on GitHub: https://github.com/micro/go-micro/blob/master/CONTRIBUTING.md
