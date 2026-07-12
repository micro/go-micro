# Go Micro on Kubernetes (alpha)

An opt-in, **alpha** foundation for running Go Micro **Agents**, **Services**,
and **Flows** on Kubernetes as first-class custom resources.

> Status: `v1alpha1`. This is the CRD + mapping foundation, not a running
> operator yet. The resource shape and the resource → Deployment mapping may
> change. Nothing here runs by default or affects existing runtime behavior.

## What's here

| Piece | Path | What it is |
|-------|------|------------|
| CRD manifests | [`crds/`](crds/) | `Agent`, `Service`, `Flow` CustomResourceDefinitions (group `micro.go-micro.dev`, version `v1alpha1`) |
| Mapping package | [`kubernetes.go`](kubernetes.go) | `Render(Resource)` → a `Deployment` (+ `Service` when a port is set), wired to the go-micro registry |
| Manifest types | [`manifest.go`](manifest.go) | The minimal typed Deployment/Service subset `Render` emits — no `client-go`/`k8s.io/api` dependency |

`Render` is a **pure function**: resource in, manifest structs out. It never
talks to a cluster, so the same mapping can back a manifest generator, a
dry-run/diff test, or (later) a real reconciler without changing.

## Custom resources

```yaml
apiVersion: micro.go-micro.dev/v1alpha1
kind: Agent           # or Service, or Flow
metadata:
  name: support
  namespace: agents
spec:
  image: example/support:v1
  replicas: 3
  registry: nats://registry:4222   # → MICRO_REGISTRY_ADDRESS in the pod
  port: 8080                        # → containerPort + a ClusterIP Service
  env:
    LOG_LEVEL: debug
  args: ["run"]
```

A resource maps to:

- a **Deployment** (`apps/v1`) with the standard labels
  (`app.kubernetes.io/name`, `app.kubernetes.io/managed-by: go-micro`,
  `micro.go-micro.dev/kind: <Agent|Service|Flow>`), the desired replicas
  (default `1`), and a container whose env carries `MICRO_REGISTRY_ADDRESS`
  (when `registry` is set) and `MICRO_KIND`, followed by your `env`;
- a **Service** (`v1`, ClusterIP) — only when `spec.port > 0`.

## Local validation

Register the resource types with any cluster (kind, minikube, k3d, real):

```bash
kubectl apply -f deploy/kubernetes/crds/
kubectl get crds | grep micro.go-micro.dev
```

Generate manifests from a resource in Go:

```go
r := kubernetes.Resource{
    Kind:     kubernetes.KindAgent,
    Metadata: kubernetes.Metadata{Name: "support"},
    Spec:     kubernetes.Spec{Image: "example/support:v1", Port: 8080, Registry: "nats://registry:4222"},
}
rendered, _ := kubernetes.Render(r)
manifest, _ := rendered.YAML()   // apply with: kubectl apply -f -
```

The mapping and the CRD manifests are covered by `go test ./deploy/kubernetes/...`
(the CRDs are validated structurally in CI).

## Roadmap

- A reconciler (controller) that watches these resources and applies the mapped
  objects — kept behind this same `Render` seam so the mapping stays testable
  without a cluster.
- Wiring `Flow` to durable-workflow execution, and `Agent` to the A2A/MCP
  gateways, once those land.
