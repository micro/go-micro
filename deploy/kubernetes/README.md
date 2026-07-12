# Kubernetes deployment foundation (alpha)

This package is the first opt-in Kubernetes foundation for the Go Micro lifecycle:
`Service`, `Agent`, and `Flow` resources. It is intentionally experimental and
additive. Nothing in the Go Micro runtime installs these resources or changes
production defaults.

## What is included

- Alpha CRD manifests in `config/crd/` for `agents.micro.dev`,
  `services.micro.dev`, and `flows.micro.dev`.
- A small dependency-free mapper that turns a desired Go Micro resource into the
  Kubernetes `Deployment` shape an operator reconciliation loop will own.
- Unit tests that validate the structural CRD fragments and dry-run the
  Agent-to-Deployment mapping.

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
