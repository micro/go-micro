package kubernetes

import (
	"fmt"
	"sort"
	"strings"
)

const (
	// Group is the API group for the alpha Go Micro Kubernetes resources.
	Group = "micro.dev"
	// Version is the current alpha API version for the CRDs in this package.
	Version = "v1alpha1"
)

// Kind identifies a Go Micro lifecycle resource that can be reconciled toward a
// Kubernetes Deployment.
type Kind string

const (
	KindAgent   Kind = "Agent"
	KindService Kind = "Service"
	KindFlow    Kind = "Flow"
)

// WorkloadSpec is the common alpha spec shared by Agent, Service, and Flow CRDs.
type WorkloadSpec struct {
	Image       string            `json:"image"`
	Command     []string          `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Replicas    int32             `json:"replicas,omitempty"`
	Registry    string            `json:"registry,omitempty"`
	Environment map[string]string `json:"env,omitempty"`
}

// Resource is the minimal desired state for a Go Micro lifecycle resource.
type Resource struct {
	Kind      Kind
	Name      string
	Namespace string
	Spec      WorkloadSpec
}

// Deployment is a small, dependency-free representation of the Kubernetes
// Deployment fields the alpha reconciler skeleton owns.
type Deployment struct {
	Name      string
	Namespace string
	Labels    map[string]string
	Replicas  int32
	Pod       PodTemplate
}

// PodTemplate describes the pod fields emitted by MapDeployment.
type PodTemplate struct {
	Labels    map[string]string
	Container Container
}

// Container describes the single Go Micro workload container.
type Container struct {
	Name        string
	Image       string
	Command     []string
	Args        []string
	Environment map[string]string
}

// MapDeployment maps a Go Micro alpha resource to the Deployment shape an
// operator reconciliation loop would apply.
func MapDeployment(resource Resource) (Deployment, error) {
	if resource.Kind != KindAgent && resource.Kind != KindService && resource.Kind != KindFlow {
		return Deployment{}, fmt.Errorf("unsupported kind %q", resource.Kind)
	}
	name := strings.TrimSpace(resource.Name)
	if name == "" {
		return Deployment{}, fmt.Errorf("name is required")
	}
	image := strings.TrimSpace(resource.Spec.Image)
	if image == "" {
		return Deployment{}, fmt.Errorf("spec.image is required")
	}

	namespace := strings.TrimSpace(resource.Namespace)
	if namespace == "" {
		namespace = "default"
	}
	replicas := resource.Spec.Replicas
	if replicas == 0 {
		replicas = 1
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       name,
		"app.kubernetes.io/managed-by": "go-micro",
		"micro.dev/kind":               strings.ToLower(string(resource.Kind)),
	}
	env := copyMap(resource.Spec.Environment)
	if resource.Spec.Registry != "" {
		env["MICRO_REGISTRY"] = resource.Spec.Registry
	}

	return Deployment{
		Name:      name,
		Namespace: namespace,
		Labels:    copyMap(labels),
		Replicas:  replicas,
		Pod: PodTemplate{
			Labels: copyMap(labels),
			Container: Container{
				Name:        name,
				Image:       image,
				Command:     append([]string(nil), resource.Spec.Command...),
				Args:        append([]string(nil), resource.Spec.Args...),
				Environment: env,
			},
		},
	}, nil
}

// EnvironmentKeys returns stable environment variable keys from a mapped
// container. It is useful for deterministic validation and rendering.
func (c Container) EnvironmentKeys() []string {
	keys := make([]string, 0, len(c.Environment))
	for key := range c.Environment {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func copyMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
