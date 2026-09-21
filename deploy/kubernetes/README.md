# Kubernetes deployment foundation (alpha)

This package is the first opt-in Kubernetes foundation for the Go Micro lifecycle:
`Service`, `Agent`, and `Flow` resources. It is intentionally experimental and
additive. Nothing in the Go Micro runtime installs these resources or changes
production defaults.

## What is included

- Alpha CRD manifests in `config/crd/` for `agents.micro.dev`,
  `services.micro.dev`, and `flows.micro.dev`.
- A small dependency-free mapper that turns a desired Go Micro resource into the
  Kubernetes `Deployment` and ClusterIP `Service` shapes a reconciliation loop owns.
- A dependency-free `Reconcile(desired, observed)` core that decides the one
  action needed to converge (create / update / noop) and the `Ready`/`Error`
  status conditions — no controller-runtime, no client-go, fully unit-testable.
  `ReconcileWorkloads` renders both Kubernetes API objects.
- An opt-in `Controller` that observes workloads, applies changes and writes CR
  status through a `WorkloadClient` adapter. CRDs enable the status subresource.
  A host supplies its Kubernetes client, watches and requeue policy.
- Unit tests that validate the structural CRD fragments, the Agent-to-Deployment
  mapping, controller create/update/noop behavior, and error status propagation.

## Local validation

```sh
go test ./deploy/kubernetes
```

If you have a Kubernetes cluster and `kubectl` available, you can also perform a
server-side dry run of the CRDs:

```sh
kubectl apply --dry-run=server -f deploy/kubernetes/config/crd/
```

The manifests are `v1alpha1`; expect the API shape to evolve before this becomes
a production operator.

## Controller integration

```go
controller := kubernetes.Controller{Client: adapter}
conditions, err := controller.Reconcile(ctx, kubernetes.Resource{
    Kind: kubernetes.KindService,
    Name: "hello",
    Namespace: "default",
    UID: objectUID,
    Spec: kubernetes.WorkloadSpec{Image: "example/hello:v1", Port: 8080},
})
```

`adapter` implements `WorkloadClient`: observe the managed Deployment/Service
fields, apply the supplied API objects with a stable server-side-apply field
manager, and replace `status.conditions` on the source custom resource. Exclude
server-assigned Service addresses and unmanaged fields from observations. The
source UID adds owner references so Kubernetes can garbage-collect workloads.

The default RPC port is 8080. The mapper supplies `MICRO_SERVER_ADDRESS` unless
explicitly set; an override must listen on `spec.port`. Names must be unique
across Agent, Service and Flow resources within a namespace because they share
the Deployment and Service namespace. Ready remains false while either object
changes or the desired replica count is unavailable. Adapters must observe the
current Deployment generation before reporting ready replicas.

This is a controller skeleton, not an installed operator: there is no bundled
cluster client, watch loop, RBAC installer or production rollout policy. A host
must arrange reconciliation on resource/workload updates and retry errors.
