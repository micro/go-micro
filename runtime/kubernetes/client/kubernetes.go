// Package client provides an implementation of a restricted subset of kubernetes API client
package client

// Kubernetes client
type Kubernetes interface {
	// CreateDeployment creates new kubernetes deployment
	CreateDeployment(*Deployment) error
	// GetDeployment queries kubernetes deployments and returns the matches
	GetDeployment(map[string]string) (*DeploymentList, error)
	// UpdateDeployment patches deployment annotations with new metadata
	UpdateDeployment(*Deployment) error
	// ListDeployments lists all micro service deployments
	ListDeployments() (*DeploymentList, error)
	// DeleteDeployment deletes kubernetes deployment
	DeleteDeployment(*Deployment) error
	// CreateService creates new kubernetes service
	CreateService(*Service) error
	// GetService queries kubernetes services and returns the matches
	GetService(map[string]string) (*ServiceList, error)
	// UpdateService updates kubernetes service
	UpdateService(*Service) error
	// DeleteService deletes kubernetes service
	DeleteService(*Service) error
	// ListServices lists all micro services running in Kubernetes
	ListServices() (*ServiceList, error)
}

// ServicePort configures service ports
type ServicePort struct {
	Name string `json:"name,omitempty"`
	Port int    `json:"port"`
}

// ServiceSpec provides service configuration
type ServiceSpec struct {
	Ports    []ServicePort     `json:"ports,omitempty"`
	Selector map[string]string `json:"selector,omitempty"`
	Type     string            `json:"type,omitempty"`
}

// ServiceStatus
type ServiceStatus struct{}

// Service is kubernetes service
type Service struct {
	Metadata *Metadata      `json:"metadata"`
	Spec     *ServiceSpec   `json:"spec,omitempty"`
	Status   *ServiceStatus `json:"status,omitempty"`
}

// ServiceList
type ServiceList struct {
	Items []Service `json:"items"`
}

// Metadata defines api request metadata
type Metadata struct {
	Name        string            `json:"name,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	Version     string            `json:"version,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type ContainerPort struct {
	Name          string `json:"name,omitempty"`
	HostPort      int    `json:"hostPort,omitempty"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type Container struct {
	Name  string          `json:"name"`
	Image string          `json:"image,omitempty"`
	Env   []EnvVar        `json:"env,omitempty"`
	Ports []ContainerPort `json:"ports,omitempty"`
}

// PodSpec
type PodSpec struct {
	Containers []Container `json:"containers"`
}

// Template is micro deployment template
type Template struct {
	Metadata *Metadata `json:"metadata,omitempty"`
	PodSpec  *PodSpec  `json:"spec,omitempty"`
}

// LabelSelector is a label query over a set of resources
// NOTE: we do not support MatchExpressions at the moment
type LabelSelector struct {
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

// DeploymentSpec defines micro deployment spec
type DeploymentSpec struct {
	Replicas int            `json:"replicas,omitempty"`
	Selector *LabelSelector `json:"selector"`
	Template *Template      `json:"template,omitempty"`
}

// DeploymentStatus is returned when querying deployment
type DeploymentStatus struct {
	Replicas            int `json:"replicas,omitempty"`
	UpdatedReplicas     int `json:"updatedReplicas,omitempty"`
	ReadyReplicas       int `json:"readyReplicas,omitempty"`
	AvailableReplicas   int `json:"availableReplicas,omitempty"`
	UnavailableReplicas int `json:"unavailableReplicas,omitempty"`
}

// Deployment is Kubernetes deployment
type Deployment struct {
	Metadata *Metadata         `json:"metadata"`
	Spec     *DeploymentSpec   `json:"spec,omitempty"`
	Status   *DeploymentStatus `json:"status,omitempty"`
}

// DeploymentList
type DeploymentList struct {
	Items []Deployment `json:"items"`
}
