# Kubernetes Registry

A Go Micro registry that uses Kubernetes as the service registry.

```go
import (
    "go-micro.dev/v6"
    "go-micro.dev/v6/registry/kubernetes"
)

svc := micro.NewService("greeter", micro.Registry(kubernetes.NewRegistry()))
```

To select it by flag or environment (`--registry kubernetes`,
`MICRO_REGISTRY=kubernetes`), blank-import `go-micro.dev/v6/cmd/defaults`.

It reads and writes the same pod labels and annotations as the
`github.com/micro/plugins/v5/registry/kubernetes` plugin, so v5 and v6 services
discover each other during an upgrade.


## Overview
This registry plugin makes use of Annotations and Labels on a Kubernetes pod
to build a service discovery mechanism.


## RBAC
If your Kubernetes cluster has RBAC enabled, a role and role binding
will need to be created to allow this plugin to `list` and `patch` pods.

A cluster role can be used to specify the `list` and `patch`
requirements, while a role binding per namespace can be used to apply
the cluster role. The example RBAC configs below assume your Micro-based
services are running in the `test` namespace, and the pods that contain
the services are using the `micro-services` service account.

```
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: micro-registry
rules:
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - list
  - patch
  - watch
```

```
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: micro-registry
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: micro-registry
subjects:
- kind: ServiceAccount
  name: micro-services
  namespace: test
```


## Gotchas
* Registering/Deregistering relies on the HOSTNAME Environment Variable, which inside a pod
is the place where it can be retrieved from. (This needs improving)


## Connecting to the Kubernetes API
### Within a pod
If the `--registry_address` flag (`MICRO_REGISTRY_ADDRESS`) is omitted, the plugin will securely connect to
the Kubernetes API using the pods "Service Account". No extra configuration is necessary.

Find out more about service accounts here. http://kubernetes.io/docs/user-guide/accessing-the-cluster/

### Outside of Kubernetes
Some functions of the plugin should work, but its not been heavily tested.
Currently no TLS support.
