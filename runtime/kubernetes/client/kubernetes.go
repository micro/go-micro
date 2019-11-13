// Package client provides an implementation of a restricted subset of kubernetes API client
package client

import (
	"strconv"
	"time"

	"github.com/micro/go-micro/util/log"
)

var (
	// DefaultImage is default micro image
	DefaultImage = "micro/micro"
)

// Kubernetes client
type Kubernetes interface {
	// CreateDeployment creates new kubernetes deployment
	CreateDeployment(*Deployment) error
	// GetDeployment queries kubernetes deployments and returns the matches
	GetDeployment(map[string]string) (*DeploymentList, error)
	// UpdateDeployment patches deployment annotations with new metadata
	UpdateDeployment(*Deployment) error
	// DeleteDeployment deletes kubernetes deployment
	DeleteDeployment(*Deployment) error
	// ListDeployments lists all micro service deployments
	ListDeployments() (*DeploymentList, error)
	// CreateService creates new kubernetes service
	CreateService(*Service) error
	// GetService queries kubernetes services and returns the matches
	GetService(map[string]string) (*ServiceList, error)
	// UpdateService patches kubernetes service
	UpdateService(*Service) error
	// DeleteService deletes kubernetes service
	DeleteService(*Service) error
	// ListServices lists all micro services running in Kubernetes
	ListServices() (*ServiceList, error)
}

// DefaultService returns default micro kubernetes service definition
func DefaultService(name, version string) *Service {
	Labels := map[string]string{
		"name":    name,
		"micro":   "service",
		"version": version,
	}

	Metadata := &Metadata{
		Name:      name,
		Namespace: "default",
		Version:   version,
		Labels:    Labels,
	}

	Spec := &ServiceSpec{
		Type:     "ClusterIP",
		Selector: Labels,
		Ports: []ServicePort{{
			name + "-port", 9090, "",
		}},
	}

	return &Service{
		Metadata: Metadata,
		Spec:     Spec,
	}
}

// DefaultService returns default micro kubernetes deployment definition
func DefaultDeployment(name, version string) *Deployment {
	Labels := map[string]string{
		"name":    name,
		"micro":   "service",
		"version": version,
	}

	Metadata := &Metadata{
		Name:      name,
		Namespace: "default",
		Version:   version,
		Labels:    Labels,
	}

	// TODO: we need to figure out this version stuff; might need to add Build to runtime.Service
	buildTime, err := strconv.ParseInt(version, 10, 64)
	if err == nil {
		buildUnixTimeUTC := time.Unix(buildTime, 0)
		Metadata.Annotations = map[string]string{
			"build": buildUnixTimeUTC.Format(time.RFC3339),
		}
	} else {
		log.Debugf("Runtime could not parse build: %v", err)
	}

	Env := []EnvVar{
		{"MICRO_BROKER", "nats"},
		{"MICRO_BROKER_ADDRESS", "nats-cluster"},
		{"MICRO_REGISTRY", "etcd"},
		{"MICRO_REGISTRY_ADDRESS", "etcd-cluster"},
		{"MICRO_PROXY", "go.micro.proxy"},
	}

	// TODO: change the image name here
	Spec := &DeploymentSpec{
		Replicas: 1,
		Selector: &LabelSelector{
			MatchLabels: Labels,
		},
		Template: &Template{
			Metadata: Metadata,
			PodSpec: &PodSpec{
				Containers: []Container{{
					Name:    name,
					Image:   DefaultImage,
					Env:     Env,
					Command: []string{"go", "run", "main.go"},
					Ports: []ContainerPort{{
						Name:          name + "-port",
						ContainerPort: 8080,
					}},
				}},
			},
		},
	}

	return &Deployment{
		Metadata: Metadata,
		Spec:     Spec,
	}
}
