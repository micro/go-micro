// Package client provides an implementation of a restricted subset of kubernetes API client
package client

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/micro/go-micro/util/log"
)

const (
	// https://github.com/kubernetes/apimachinery/blob/master/pkg/util/validation/validation.go#L134
	dns1123LabelFmt string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
)

var (
	// DefaultImage is default micro image
	DefaultImage = "micro/micro"
	// ServiceRegexp is used to validate service name
	ServiceRegexp = regexp.MustCompile("^" + dns1123LabelFmt + "$")
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

// DefaultService returns default micro kubernetes service definition
func DefaultService(name, version string) *Service {
	Labels := map[string]string{
		"name":    name,
		"version": version,
		"micro":   "service",
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

// DefaultService returns default micro kubernetes deployment definition
func DefaultDeployment(name, version string) *Deployment {
	Labels := map[string]string{
		"name":    name,
		"version": version,
		"micro":   "service",
	}

	// API deployment object name joins name and version over "="
	depName := strings.Join([]string{name, version}, "-")

	Metadata := &Metadata{
		Name:      depName,
		Namespace: "default",
		Version:   version,
		Labels:    Labels,
	}

	// TODO: we need to figure out this version stuff
	// might be worth adding Build to runtime.Service
	buildTime, err := strconv.ParseInt(version, 10, 64)
	if err == nil {
		buildUnixTimeUTC := time.Unix(buildTime, 0)
		Metadata.Annotations = map[string]string{
			"build": buildUnixTimeUTC.Format(time.RFC3339),
		}
	} else {
		log.Debugf("Runtime could not parse build: %v", err)
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
					Env:     []EnvVar{},
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
