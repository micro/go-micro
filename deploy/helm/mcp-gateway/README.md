# MCP Gateway Helm Chart

Deploy the Go Micro MCP Gateway on Kubernetes. The gateway discovers go-micro services via a registry and exposes them as AI-accessible tools through the Model Context Protocol.

## Quick Start

```bash
helm install mcp-gateway ./deploy/helm/mcp-gateway \
  --set gateway.registry=consul \
  --set gateway.registryAddress=consul:8500
```

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of gateway replicas | `1` |
| `image.repository` | Container image | `ghcr.io/micro/mcp-gateway` |
| `image.tag` | Image tag (defaults to appVersion) | `""` |
| `gateway.address` | Listen address | `:3000` |
| `gateway.registry` | Registry backend (mdns, consul, etcd) | `consul` |
| `gateway.registryAddress` | Registry address | `consul:8500` |
| `gateway.rateLimit` | Requests/second per tool (0=unlimited) | `0` |
| `gateway.rateBurst` | Rate limit burst size | `20` |
| `gateway.auth` | Enable JWT authentication | `false` |
| `gateway.audit` | Enable audit logging | `false` |
| `gateway.scopes` | Per-tool scope requirements | `[]` |
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.port` | Service port | `3000` |
| `ingress.enabled` | Enable ingress | `false` |
| `autoscaling.enabled` | Enable HPA | `false` |
| `autoscaling.minReplicas` | Minimum replicas | `1` |
| `autoscaling.maxReplicas` | Maximum replicas | `10` |

## Examples

### Production with Consul

```bash
helm install mcp-gateway ./deploy/helm/mcp-gateway \
  --set replicaCount=3 \
  --set gateway.registry=consul \
  --set gateway.registryAddress=consul.default.svc:8500 \
  --set gateway.auth=true \
  --set gateway.audit=true \
  --set gateway.rateLimit=100 \
  --set autoscaling.enabled=true
```

### With Ingress (nginx)

```bash
helm install mcp-gateway ./deploy/helm/mcp-gateway \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.hosts[0].host=mcp.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix \
  --set ingress.tls[0].secretName=mcp-tls \
  --set ingress.tls[0].hosts[0]=mcp.example.com
```

### With Scopes

```bash
helm install mcp-gateway ./deploy/helm/mcp-gateway \
  --set gateway.auth=true \
  --set 'gateway.scopes[0]=blog.Blog.Create=blog:write' \
  --set 'gateway.scopes[1]=blog.Blog.Delete=blog:admin'
```

## Architecture

```
                         Kubernetes Cluster
  ┌──────────────────────────────────────────────────────────┐
  │                                                          │
  │  ┌─────────┐   MCP    ┌─────────────┐   RPC   ┌──────┐ │
  │  │ Ingress │ ───────> │ MCP Gateway │ ──────> │ Svc  │ │
  │  │         │          │  (N pods)   │         │ Pods │ │
  │  └─────────┘          └─────────────┘         └──────┘ │
  │                             │                    │      │
  │                             v                    v      │
  │                        ┌──────────┐                     │
  │                        │  Consul  │                     │
  │                        │ Registry │                     │
  │                        └──────────┘                     │
  └──────────────────────────────────────────────────────────┘
```
