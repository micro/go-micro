# Kubernetes ConfigMap Source (configmap)

The configmap source reads config from a kubernetes configmap key/values

## Kubernetes ConfigMap Format

The configmap source expects keys under a namespace default to `default` and a confimap default to `micro`

```shell
// we recommend to setup your variables from multiples files example:
$ kubectl create configmap micro --namespace default --from-file=./testdata

// verify if were set correctly with
$ kubectl get configmap micro --namespace default
{
    "apiVersion": "v1",
    "data": {
        "config": "host=0.0.0.0\nport=1337",
        "mongodb": "host=127.0.0.1\nport=27017\nuser=user\npassword=password",
        "redis": "url=redis://127.0.0.1:6379/db01"
    },
    "kind": "ConfigMap",
    "metadata": {
        ...
        "name": "micro",
        "namespace": "default",
        ...
    }
}
```

Keys are split on `\n` and `=` this is because the way kubernetes saves the data is `map[string][string]`.

```go
// the example above "mongodb": "host=127.0.0.1\nport=27017\nuser=user\npassword=password" will be accessible as:
conf.Get("mongodb", "host") // 127.0.0.1
conf.Get("mongodb", "port") // 27017
```

## Kubernetes wrights

Since Kubernetes 1.9 the app must have wrights to be able to access configmaps. You must provide Role and RoleBinding so that your app can access configmaps.

```yaml
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: api-role
  labels:
    app: tools-rbac
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "update", "list", "watch"]
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: global-rolebinding
  labels:
    app: tools-rbac
subjects:
- kind: Group
  name: system:serviceaccounts
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: api-role
  apiGroup: ""
```
To configure your Kubernetes cluster just apply the file:
```bash
kubectl apply -n YourNameSpace -f role.yaml
```

## New Source

Specify source with data

```go
configmapSource := configmap.NewSource(
	// optionally specify a namespace; default to default
	configmap.WithNamespace("kube-public"),
	// optionally specify name for ConfigMap; defaults micro
	configmap.WithName("micro-config"),
    // optionally strip the provided path to a kube config file mostly used outside of a cluster, defaults to "" for in cluster support.
    configmap.WithConfigPath($HOME/.kube/config),
)
```

## Load Source

Load the source into config

```go
// Create new config
conf := config.NewConfig()

// Load file source
conf.Load(configmapSource)
```

## Running Go Tests

### Requirements

Have a kubernetes cluster running (external or minikube) have a valid `kubeconfig` file.

```shell
// Setup testing configmaps feel free to remove them after testing.
$ cd source/configmap
$ kubectl create configmap micro --from-file=./testdata
$ kubectl create configmap micro --from-file=./testdata --namespace kube-public
$ kubectl create configmap micro-config --from-file=./testdata
$ kubectl create configmap micro-config --from-file=./testdata --namespace kube-public
$ go test -v -cover
```

```shell
// To clean up the testing configmaps
$ kubectl delete configmap micro --all-namespaces
$ kubectl delete configmap micro-config --all-namespaces
```

## Todos
- [ ] add more test cases including watchers
- [ ] add support for prefixing either using namespace or a custom `string` passed as `WithPrefix`
- [ ] a better way to test without manual setup from the user.
- [ ] add test examples.
- [ ] open to suggestions and feedback please let me know what else should I add.

**stay tuned for kubernetes secret support as an source.**