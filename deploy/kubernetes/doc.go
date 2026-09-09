// Package kubernetes contains the experimental Kubernetes deployment foundation
// for Go Micro services, agents, and flows.
//
// The package is intentionally small and additive: it exposes alpha custom
// resource manifests and a dry-run mapper that turns a resource spec into the
// Deployment shape an operator would reconcile. It does not install an operator
// or change any runtime defaults.
package kubernetes
