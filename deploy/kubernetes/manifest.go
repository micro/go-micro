package kubernetes

// This file holds the minimal typed subset of the Kubernetes Deployment and
// Service manifests that Render emits. They are hand-written (rather than
// pulled from k8s.io/api) to keep this package free of the client-go/api
// dependency tree — the fields here are only those Render sets, in the order
// kubectl users expect to read them.

// ObjectMeta is the metadata common to the emitted objects.
type ObjectMeta struct {
	Name      string            `yaml:"name,omitempty" json:"name,omitempty"`
	Namespace string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Labels    map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// LabelSelector selects pods by label.
type LabelSelector struct {
	MatchLabels map[string]string `yaml:"matchLabels" json:"matchLabels"`
}

// EnvVar is a container environment variable.
type EnvVar struct {
	Name  string `yaml:"name" json:"name"`
	Value string `yaml:"value" json:"value"`
}

// ContainerPort exposes a port on a container.
type ContainerPort struct {
	ContainerPort int32 `yaml:"containerPort" json:"containerPort"`
}

// Container is a single workload container.
type Container struct {
	Name  string          `yaml:"name" json:"name"`
	Image string          `yaml:"image" json:"image"`
	Args  []string        `yaml:"args,omitempty" json:"args,omitempty"`
	Ports []ContainerPort `yaml:"ports,omitempty" json:"ports,omitempty"`
	Env   []EnvVar        `yaml:"env,omitempty" json:"env,omitempty"`
}

// PodSpec is the pod's container set.
type PodSpec struct {
	Containers []Container `yaml:"containers" json:"containers"`
}

// PodTemplateSpec is the pod template embedded in a Deployment.
type PodTemplateSpec struct {
	Metadata ObjectMeta `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	Spec     PodSpec    `yaml:"spec" json:"spec"`
}

// DeploymentSpec is the desired state of a Deployment.
type DeploymentSpec struct {
	Replicas int32           `yaml:"replicas" json:"replicas"`
	Selector LabelSelector   `yaml:"selector" json:"selector"`
	Template PodTemplateSpec `yaml:"template" json:"template"`
}

// Deployment is a Kubernetes apps/v1 Deployment.
type Deployment struct {
	APIVersion string         `yaml:"apiVersion" json:"apiVersion"`
	Kind       string         `yaml:"kind" json:"kind"`
	Metadata   ObjectMeta     `yaml:"metadata" json:"metadata"`
	Spec       DeploymentSpec `yaml:"spec" json:"spec"`
}

// ServicePort maps a Service port to a target container port.
type ServicePort struct {
	Port       int32 `yaml:"port" json:"port"`
	TargetPort int32 `yaml:"targetPort" json:"targetPort"`
}

// ServiceSpec is the desired state of a Service.
type ServiceSpec struct {
	Selector map[string]string `yaml:"selector" json:"selector"`
	Ports    []ServicePort     `yaml:"ports" json:"ports"`
}

// Service is a Kubernetes v1 Service (ClusterIP by default).
type Service struct {
	APIVersion string      `yaml:"apiVersion" json:"apiVersion"`
	Kind       string      `yaml:"kind" json:"kind"`
	Metadata   ObjectMeta  `yaml:"metadata" json:"metadata"`
	Spec       ServiceSpec `yaml:"spec" json:"spec"`
}
