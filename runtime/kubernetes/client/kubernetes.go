package client

// Kubernetes client
type Kubernetes interface {
	// UpdateDeployment patches deployment annotations with new metadata
	UpdateDeployment(string, interface{}) error
	// ListDeployments lists all micro deployments
	ListDeployments(labels map[string]string) (*DeploymentList, error)
}

// Template is micro deployment template
type Template struct {
	Metadata *Metadata `json:"metadata,omitempty"`
}

// Spec defines micro deployment spec
type Spec struct {
	Template *Template `json:"template,omitempty"`
}

// Metadata defines api request metadata
type Metadata struct {
	Name        string            `json:"name,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// DeploymentList
type DeploymentList struct {
	Items []Deployment `json:"items"`
}

// Deployment is Kubernetes deployment
type Deployment struct {
	Metadata *Metadata `json:"metadata"`
	Status   *Status   `json:"status"`
}

// Status is Kubernetes deployment status
type Status struct {
	Replicas          int `json:"replicas"`
	AvailableReplicas int `json:"availablereplicas"`
}
