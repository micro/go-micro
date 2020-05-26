package client

// ContainerPort
type ContainerPort struct {
	Name          string `json:"name,omitempty"`
	HostPort      int    `json:"hostPort,omitempty"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
}

// EnvVar is environment variable
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type Condition struct {
	Started string `json:"startedAt,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

// Container defined container runtime values
type Container struct {
	Name    string          `json:"name"`
	Image   string          `json:"image"`
	Env     []EnvVar        `json:"env,omitempty"`
	Command []string        `json:"command,omitempty"`
	Args    []string        `json:"args,omitempty"`
	Ports   []ContainerPort `json:"ports,omitempty"`
}

// DeploymentSpec defines micro deployment spec
type DeploymentSpec struct {
	Replicas int            `json:"replicas,omitempty"`
	Selector *LabelSelector `json:"selector"`
	Template *Template      `json:"template,omitempty"`
}

// DeploymentCondition describes the state of deployment
type DeploymentCondition struct {
	LastUpdateTime string `json:"lastUpdateTime"`
	Type           string `json:"type"`
	Reason         string `json:"reason,omitempty"`
	Message        string `json:"message,omitempty"`
}

// DeploymentStatus is returned when querying deployment
type DeploymentStatus struct {
	Replicas            int                   `json:"replicas,omitempty"`
	UpdatedReplicas     int                   `json:"updatedReplicas,omitempty"`
	ReadyReplicas       int                   `json:"readyReplicas,omitempty"`
	AvailableReplicas   int                   `json:"availableReplicas,omitempty"`
	UnavailableReplicas int                   `json:"unavailableReplicas,omitempty"`
	Conditions          []DeploymentCondition `json:"conditions,omitempty"`
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

// LabelSelector is a label query over a set of resources
// NOTE: we do not support MatchExpressions at the moment
type LabelSelector struct {
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

type LoadBalancerIngress struct {
	IP       string `json:"ip,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

type LoadBalancerStatus struct {
	Ingress []LoadBalancerIngress `json:"ingress,omitempty"`
}

// Metadata defines api object metadata
type Metadata struct {
	Name        string            `json:"name,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	Version     string            `json:"version,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PodSpec is a pod
type PodSpec struct {
	Containers         []Container `json:"containers"`
	ServiceAccountName string      `json:"serviceAccountName"`
}

// PodList
type PodList struct {
	Items []Pod `json:"items"`
}

// Pod is the top level item for a pod
type Pod struct {
	Metadata *Metadata  `json:"metadata"`
	Spec     *PodSpec   `json:"spec,omitempty"`
	Status   *PodStatus `json:"status"`
}

// PodStatus
type PodStatus struct {
	Conditions []PodCondition    `json:"conditions,omitempty"`
	Containers []ContainerStatus `json:"containerStatuses"`
	PodIP      string            `json:"podIP"`
	Phase      string            `json:"phase"`
	Reason     string            `json:"reason"`
}

// PodCondition describes the state of pod
type PodCondition struct {
	Type    string `json:"type"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type ContainerStatus struct {
	State ContainerState `json:"state"`
}

type ContainerState struct {
	Running    *Condition `json:"running"`
	Terminated *Condition `json:"terminated"`
	Waiting    *Condition `json:"waiting"`
}

// Resource is API resource
type Resource struct {
	Name  string
	Kind  string
	Value interface{}
}

// ServicePort configures service ports
type ServicePort struct {
	Name     string `json:"name,omitempty"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol,omitempty"`
}

// ServiceSpec provides service configuration
type ServiceSpec struct {
	ClusterIP string            `json:"clusterIP"`
	Type      string            `json:"type,omitempty"`
	Selector  map[string]string `json:"selector,omitempty"`
	Ports     []ServicePort     `json:"ports,omitempty"`
}

// ServiceStatus
type ServiceStatus struct {
	LoadBalancer LoadBalancerStatus `json:"loadBalancer,omitempty"`
}

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

// Template is micro deployment template
type Template struct {
	Metadata *Metadata `json:"metadata,omitempty"`
	PodSpec  *PodSpec  `json:"spec,omitempty"`
}

// Namespace is a Kubernetes Namespace
type Namespace struct {
	Metadata *Metadata `json:"metadata,omitempty"`
}

// NamespaceList
type NamespaceList struct {
	Items []Namespace `json:"items"`
}

// ImagePullSecret
type ImagePullSecret struct {
	Name string `json:"name"`
}

// Secret
type Secret struct {
	Type     string            `json:"type,omitempty"`
	Data     map[string]string `json:"data"`
	Metadata *Metadata         `json:"metadata"`
}

// ServiceAccount
type ServiceAccount struct {
	Metadata         *Metadata         `json:"metadata,omitempty"`
	ImagePullSecrets []ImagePullSecret `json:"imagePullSecrets,omitempty"`
}
