// Package client provides an implementation of a restricted subset of kubernetes API client
package client

import (
	"strings"

	"github.com/micro/go-micro/util/log"
)

var (
	// DefaultImage is default micro image
	DefaultImage = "micro/go-micro"
)

// Kubernetes client
type Kubernetes interface {
	// Create creates new API resource
	Create(*Resource) error
	// Get queries API resrouces
	Get(*Resource, map[string]string) error
	// Update patches existing API object
	Update(*Resource) error
	// Delete deletes API resource
	Delete(*Resource) error
	// List lists API resources
	List(*Resource) error
}

// NewService returns default micro kubernetes service definition
func NewService(name, version, typ string) *Service {
	log.Tracef("kubernetes default service: name: %s, version: %s", name, version)

	Labels := map[string]string{
		"name":    name,
		"version": version,
		"micro":   typ,
	}

	svcName := name
	if len(version) > 0 {
		// API service object name joins name and version over "-"
		svcName = strings.Join([]string{name, version}, "-")
	}

	Metadata := &Metadata{
		Name:      svcName,
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

// NewService returns default micro kubernetes deployment definition
func NewDeployment(name, version, typ string) *Deployment {
	log.Tracef("kubernetes default deployment: name: %s, version: %s", name, version)

	Labels := map[string]string{
		"name":    name,
		"version": version,
		"micro":   typ,
	}

	depName := name
	if len(version) > 0 {
		// API deployment object name joins name and version over "-"
		depName = strings.Join([]string{name, version}, "-")
	}

	Metadata := &Metadata{
		Name:        depName,
		Namespace:   "default",
		Version:     version,
		Labels:      Labels,
		Annotations: map[string]string{},
	}

	// enable go modules by default
	env := EnvVar{
		Name:  "GO111MODULE",
		Value: "on",
	}

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
					Env:     []EnvVar{env},
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
