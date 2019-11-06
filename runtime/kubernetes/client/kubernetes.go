package client

// Kubernetes client
type Kubernetes interface {
	// UpdateDeployment patches deployment annotations with new metadata
	UpdateDeployment(string, interface{}) error
	// ListDeployments lists all micro deployments
	ListDeployments() (*DeploymentList, error)
}

// Metadata defines api request metadata
type Metadata struct {
	Annotations map[string]string `json:"annotations,omitempty"`
}

// DeploymentList
type DeploymentList struct {
	Items []Deployment `json:"items"`
}

// Deployment is Kubernetes deployment
type Deployment struct {
	Name   string  `json:"name"`
	Status *Status `json:"status"`
}

// Status is Kubernetes deployment status
type Status struct {
	Replicas          int `json:"replicas"`
	AvailableReplicas int `json:"availablereplicas"`
}
