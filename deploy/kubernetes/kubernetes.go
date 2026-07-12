// Package kubernetes is the opt-in, alpha Kubernetes deployment foundation for
// Go Micro. It defines the Agent, Service, and Flow custom resources (the CRD
// manifests live in crds/ and are embedded here) and maps a resource to the
// Kubernetes objects that run it — a Deployment wired to the go-micro registry,
// plus a Service when the resource exposes a port.
//
// It is deliberately dependency-light: no controller-runtime, no client-go, no
// operator binary. Render is a pure function from a resource to plain manifest
// structs, so it can back a `kubectl apply -f`-style generator, a dry-run test,
// or (later) a real reconciler without changing the mapping. Nothing here runs
// by default or changes existing runtime behavior.
//
// Status: alpha (v1alpha1). The resource shape and mapping may change.
package kubernetes

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"gopkg.in/yaml.v3"
)

// API group and version for the go-micro custom resources.
const (
	Group      = "micro.go-micro.dev"
	Version    = "v1alpha1"
	APIVersion = Group + "/" + Version
)

// Resource kinds.
const (
	KindAgent   = "Agent"
	KindService = "Service"
	KindFlow    = "Flow"
)

//go:embed crds/*.yaml
var crdFS embed.FS

// CRDs returns the embedded CustomResourceDefinition manifests (one YAML
// document per resource kind), keyed by file name. These are what an operator
// applies to register the Agent/Service/Flow types with a cluster.
func CRDs() (map[string]string, error) {
	out := map[string]string{}
	err := fs.WalkDir(crdFS, "crds", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := crdFS.ReadFile(path)
		if err != nil {
			return err
		}
		out[d.Name()] = string(b)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Resource is a go-micro custom resource (Agent, Service, or Flow) as applied
// to a cluster. It is the input to Render.
type Resource struct {
	APIVersion string   `yaml:"apiVersion" json:"apiVersion"`
	Kind       string   `yaml:"kind" json:"kind"`
	Metadata   Metadata `yaml:"metadata" json:"metadata"`
	Spec       Spec     `yaml:"spec" json:"spec"`
}

// Metadata identifies a resource within a namespace.
type Metadata struct {
	Name      string            `yaml:"name" json:"name"`
	Namespace string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Labels    map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// Spec is the desired state shared by all three resource kinds.
type Spec struct {
	// Image is the container image that runs the workload. Required.
	Image string `yaml:"image" json:"image"`
	// Replicas is the desired pod count (defaults to 1 when nil).
	Replicas *int32 `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	// Registry is the go-micro registry address the workload registers with
	// and discovers peers through. Surfaced to the container as
	// MICRO_REGISTRY_ADDRESS.
	Registry string `yaml:"registry,omitempty" json:"registry,omitempty"`
	// Port, when > 0, is exposed as a containerPort and a ClusterIP Service.
	Port int32 `yaml:"port,omitempty" json:"port,omitempty"`
	// Env are extra environment variables for the container.
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	// Args are extra command-line arguments.
	Args []string `yaml:"args,omitempty" json:"args,omitempty"`
}

// Rendered is the set of Kubernetes objects a Resource maps to.
type Rendered struct {
	Deployment Deployment
	// Service is set only when the resource exposes a Port.
	Service *Service
}

// Render validates r and maps it to its Kubernetes objects. It does not talk to
// a cluster; the result is plain manifest structs a caller can marshal and
// apply, dry-run, or diff.
func Render(r Resource) (Rendered, error) {
	if err := validate(r); err != nil {
		return Rendered{}, err
	}
	name := r.Metadata.Name
	labels := mergeLabels(r)
	replicas := int32(1)
	if r.Spec.Replicas != nil {
		replicas = *r.Spec.Replicas
	}

	container := Container{
		Name:  name,
		Image: r.Spec.Image,
		Args:  r.Spec.Args,
		Env:   env(r),
	}
	if r.Spec.Port > 0 {
		container.Ports = []ContainerPort{{ContainerPort: r.Spec.Port}}
	}

	dep := Deployment{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Metadata:   ObjectMeta{Name: name, Namespace: r.Metadata.Namespace, Labels: labels},
		Spec: DeploymentSpec{
			Replicas: replicas,
			Selector: LabelSelector{MatchLabels: selector(name)},
			Template: PodTemplateSpec{
				Metadata: ObjectMeta{Labels: labels},
				Spec:     PodSpec{Containers: []Container{container}},
			},
		},
	}

	out := Rendered{Deployment: dep}
	if r.Spec.Port > 0 {
		out.Service = &Service{
			APIVersion: "v1",
			Kind:       "Service",
			Metadata:   ObjectMeta{Name: name, Namespace: r.Metadata.Namespace, Labels: labels},
			Spec: ServiceSpec{
				Selector: selector(name),
				Ports:    []ServicePort{{Port: r.Spec.Port, TargetPort: r.Spec.Port}},
			},
		}
	}
	return out, nil
}

// YAML marshals the rendered objects as a multi-document YAML manifest,
// Deployment first, then the Service when present.
func (r Rendered) YAML() (string, error) {
	docs := []any{r.Deployment}
	if r.Service != nil {
		docs = append(docs, *r.Service)
	}
	var out []byte
	for i, d := range docs {
		b, err := yaml.Marshal(d)
		if err != nil {
			return "", err
		}
		if i > 0 {
			out = append(out, []byte("---\n")...)
		}
		out = append(out, b...)
	}
	return string(out), nil
}

func validate(r Resource) error {
	switch r.Kind {
	case KindAgent, KindService, KindFlow:
	case "":
		return fmt.Errorf("kubernetes: resource kind is required (one of Agent, Service, Flow)")
	default:
		return fmt.Errorf("kubernetes: unknown resource kind %q (want Agent, Service, or Flow)", r.Kind)
	}
	if r.Metadata.Name == "" {
		return fmt.Errorf("kubernetes: resource metadata.name is required")
	}
	if r.Spec.Image == "" {
		return fmt.Errorf("kubernetes: %s/%s spec.image is required", r.Kind, r.Metadata.Name)
	}
	if r.Spec.Replicas != nil && *r.Spec.Replicas < 0 {
		return fmt.Errorf("kubernetes: %s/%s spec.replicas must be >= 0", r.Kind, r.Metadata.Name)
	}
	return nil
}

// selector is the immutable pod selector for a resource's Deployment/Service.
func selector(name string) map[string]string {
	return map[string]string{"app.kubernetes.io/name": name}
}

// mergeLabels combines the standard go-micro labels with any user labels; user
// labels never override the managed selector/kind labels.
func mergeLabels(r Resource) map[string]string {
	labels := map[string]string{}
	for k, v := range r.Metadata.Labels {
		labels[k] = v
	}
	labels["app.kubernetes.io/name"] = r.Metadata.Name
	labels["app.kubernetes.io/managed-by"] = "go-micro"
	labels["micro.go-micro.dev/kind"] = r.Kind
	return labels
}

// env builds the container environment: the registry address (so the workload
// joins the same go-micro registry), a kind marker, then user-supplied vars in
// deterministic order.
func env(r Resource) []EnvVar {
	var out []EnvVar
	if r.Spec.Registry != "" {
		out = append(out, EnvVar{Name: "MICRO_REGISTRY_ADDRESS", Value: r.Spec.Registry})
	}
	out = append(out, EnvVar{Name: "MICRO_KIND", Value: r.Kind})
	keys := make([]string, 0, len(r.Spec.Env))
	for k := range r.Spec.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, EnvVar{Name: k, Value: r.Spec.Env[k]})
	}
	return out
}
