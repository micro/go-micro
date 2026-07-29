---
layout: default
---

# Deployment Guide

This is a quick reference for deploying go-micro services. For the full guide, see the [Deployment documentation](../deployment.md).

## Workflow

```
micro run      →  Develop locally with hot reload
micro build    →  Compile production binaries
micro deploy   →  Push to a remote Linux server via SSH + systemd
micro server   →  Optional: production web dashboard with auth
```

## Quick Start

```bash
# Build binaries for Linux
micro build --os linux

# Deploy to server (builds automatically if needed)
micro deploy user@your-server
```

## First-Time Server Setup

On your server (any Linux with systemd):

```bash
curl -fsSL https://go-micro.dev/install.sh | sh
sudo micro init --server
```

This creates `/opt/micro/{bin,data,config}` and a systemd template for managing services.

## Deploy

```bash
micro deploy user@your-server
```

This builds for linux/amd64, copies binaries to `/opt/micro/bin/`, configures systemd services, and verifies they're running.

### Named Targets

Add deploy targets to `micro.mu`:

```
deploy prod
    ssh deploy@prod.example.com

deploy staging
    ssh deploy@staging.example.com
```

Then: `micro deploy prod`

## Managing Services

```bash
micro status --remote user@server       # Check status
micro logs --remote user@server         # View logs
micro logs myservice --remote user@server -f  # Follow logs
```

## Docker (Optional)

```bash
micro build --docker          # Build Docker images
micro build --docker --push   # Build and push
micro build --compose         # Generate docker-compose.yml
```

## Full Documentation

See the [Deployment documentation](../deployment.md) for complete details including SSH setup, environment variables, security best practices, and troubleshooting.
