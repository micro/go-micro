---
layout: default
---

# Deployment

The `micro build` and `micro deploy` commands help you go from development to production.

## Quick Start

```bash
# Build container images
micro build

# Deploy with docker-compose
micro deploy
```

## Building Images

### Basic Build

```bash
micro build
```

This:
1. Reads `micro.mu` (if present) to find services
2. Generates a `Dockerfile` for each service (if not present)
3. Runs `docker build` for each service

### Build Options

```bash
micro build --tag v1.0.0         # Specific tag (default: latest)
micro build --registry docker.io/myuser  # Push to registry
micro build --push               # Build and push
```

### Generated Dockerfile

If no Dockerfile exists, one is generated:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /service /app/service
EXPOSE 8080
CMD ["/app/service"]
```

Customize by creating your own `Dockerfile`.

## Generating docker-compose.yml

```bash
micro build --compose
```

Generates a `docker-compose.yml` from your `micro.mu` config:

```yaml
version: '3.8'

services:
  users:
    image: users:latest
    ports:
      - "8081:8081"
    environment:
      - MICRO_REGISTRY=mdns

  posts:
    image: posts:latest
    ports:
      - "8082:8082"
    depends_on:
      - users
    environment:
      - MICRO_REGISTRY=mdns
```

## Deploying

### Docker Compose

```bash
micro deploy
```

Runs `docker compose up -d` using the generated `docker-compose.yml`.

```bash
micro deploy --build  # Rebuild images first
```

### SSH Deploy

For simple deployments to a single server:

```bash
micro deploy --ssh user@host
```

This:
1. Creates `~/micro` on the remote host
2. Syncs your code using rsync
3. Builds each service on the remote host
4. Starts services in the background

```bash
micro deploy --ssh user@host --path /opt/myapp  # Custom remote path
```

### View Logs

After deploying:

```bash
# Docker Compose
docker compose logs -f

# SSH deploy
ssh user@host 'tail -f ~/micro/*.log'
```

## Complete Workflow

```bash
# 1. Develop locally
micro run

# 2. Build images
micro build --tag v1.0.0

# 3. Generate compose file
micro build --compose

# 4. Deploy
micro deploy
```

Or for SSH:

```bash
# 1. Develop locally
micro run

# 2. Deploy to server
micro deploy --ssh user@host
```

## Configuration

The `micro.mu` file drives both build and deploy:

```
service users
    path ./users
    port 8081

service posts
    path ./posts
    port 8082
    depends users

service web
    path ./web
    port 8089
    depends users posts
```

- `path` - Where to find the service code
- `port` - Exposed port (used in Dockerfile and compose)
- `depends` - Service dependencies (used in compose depends_on)

## Tips

1. **Version your images** - Use `--tag v1.0.0` not just `latest`
2. **Use a registry** - Push images with `--registry` for team sharing
3. **Custom Dockerfiles** - Override the generated one for complex builds
4. **SSH for simple deploys** - Great for single-server setups
5. **Compose for local prod** - Test production config locally
