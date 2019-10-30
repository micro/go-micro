package client

// Kubernetes client
type Kubernetes interface {
	// UpdateDeployment patches deployment annotations with new metadata
	UpdateDeployment(string, *Metadata) error
}

// Metadata defines api request metadata
type Metadata struct {
	Annotations map[string]string `json:"annotations,omitempty"`
}
