---
layout: default
---

# Deployment

Go produces self-contained binaries. No Docker required.

## Quick Start

```bash
# Build binaries
micro build --os linux

# Deploy to server
micro deploy --ssh user@host
```

## Building

### Basic Build

```bash
micro build
```

This builds Go binaries for all services in `micro.mu` (or the current directory) to `./bin/`.

### Cross-Compilation

```bash
micro build --os linux              # For Linux servers
micro build --os linux --arch arm64 # For ARM64 (e.g., AWS Graviton)
micro build --os darwin             # For macOS
micro build --os windows            # For Windows (.exe)
```

### Custom Output

```bash
micro build --output ./dist
```

## Deploying

### SSH Deploy

```bash
micro deploy --ssh user@host
```

This:
1. Copies `./bin/*` to the remote host (if exists)
2. Or syncs source and builds on remote
3. Restarts services

### Workflow

**Option 1: Build locally, copy binaries**

```bash
micro build --os linux          # Build for target OS
micro deploy --ssh user@host    # Copy and restart
```

**Option 2: Build on remote**

```bash
micro deploy --ssh user@host    # Syncs source, builds there
```

### Remote Structure

```
~/micro/
├── bin/           # Service binaries
│   ├── users
│   ├── posts
│   └── web
├── logs/          # Service logs
│   ├── users.log
│   ├── posts.log
│   └── web.log
└── src/           # Source (if building on remote)
```

### View Logs

```bash
ssh user@host 'tail -f ~/micro/logs/*.log'
```

## Docker (Optional)

If you prefer containers:

```bash
micro build --docker             # Build images
micro build --docker --push      # Build and push to registry
micro build --compose            # Generate docker-compose.yml
```

Then deploy with docker-compose on your server:

```bash
scp docker-compose.yml user@host:~/
ssh user@host 'docker compose up -d'
```

## Complete Example

```bash
# Development
micro new myapp
cd myapp
micro run              # Develop locally

# Build
micro build --os linux

# Deploy
micro deploy --ssh deploy@prod.example.com

# Check
ssh deploy@prod.example.com 'tail -f ~/micro/logs/*.log'
```

## Tips

1. **Cross-compile locally** - Faster than building on remote
2. **Use `--os linux`** - Most servers are Linux
3. **Single binary** - Go's strength, no runtime needed
4. **Logs in ~/micro/logs/** - Easy to tail and rotate
5. **No Docker needed** - Unless you want it
