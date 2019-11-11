package client

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/micro/go-micro/runtime/kubernetes/client/api"
	"github.com/micro/go-micro/util/log"
)

var (
	serviceAccountPath = "/var/run/secrets/kubernetes.io/serviceaccount"
	// ErrReadNamespace is returned when the names could not be read from service account
	ErrReadNamespace = errors.New("Could not read namespace from service account secret")
)

// Client ...
type client struct {
	opts *api.Options
}

// NewClientInCluster should work similarily to the official api
// NewInClient by setting up a client configuration for use within
// a k8s pod.
func NewClientInCluster() *client {
	host := "https://" + os.Getenv("KUBERNETES_SERVICE_HOST") + ":" + os.Getenv("KUBERNETES_SERVICE_PORT")

	s, err := os.Stat(serviceAccountPath)
	if err != nil {
		log.Fatal(err)
	}
	if s == nil || !s.IsDir() {
		log.Fatal(errors.New("no k8s service account found"))
	}

	token, err := ioutil.ReadFile(path.Join(serviceAccountPath, "token"))
	if err != nil {
		log.Fatal(err)
	}
	t := string(token)

	ns, err := detectNamespace()
	if err != nil {
		log.Fatal(err)
	}

	crt, err := CertPoolFromFile(path.Join(serviceAccountPath, "ca.crt"))
	if err != nil {
		log.Fatal(err)
	}

	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: crt,
			},
			DisableCompression: true,
		},
	}

	return &client{
		opts: &api.Options{
			Client:      c,
			Host:        host,
			Namespace:   ns,
			BearerToken: &t,
		},
	}
}

func detectNamespace() (string, error) {
	nsPath := path.Join(serviceAccountPath, "namespace")

	// Make sure it's a file and we can read it
	if s, e := os.Stat(nsPath); e != nil {
		return "", e
	} else if s.IsDir() {
		return "", ErrReadNamespace
	}

	// Read the file, and cast to a string
	if ns, e := ioutil.ReadFile(nsPath); e != nil {
		return string(ns), e
	} else {
		return string(ns), nil
	}
}

// CreateDeployment creates kubernetes deployment
func (c *client) CreateDeployment(d *Deployment) error {
	return nil
}

// GetDeployment queries deployments with given labels and returns them
func (c *client) GetDeployment(labels map[string]string) (*DeploymentList, error) {
	return nil, nil
}

// UpdateDeployment patches kubernetes deployment with metadata provided in body
func (c *client) UpdateDeployment(d *Deployment) error {
	return api.NewRequest(c.opts).
		Patch().
		Resource("deployments").
		Name(d.Metadata.Name).
		Body(d.Spec).
		Do().
		Error()
}

// ListDeployments lists all kubernetes deployments with given labels
func (c *client) ListDeployments() (*DeploymentList, error) {
	// TODO: this list all micro services
	labels := map[string]string{
		"micro": "service",
	}

	var deployments DeploymentList
	err := api.NewRequest(c.opts).
		Get().
		Resource("deployments").
		Params(&api.Params{LabelSelector: labels}).
		Do().
		Into(&deployments)

	return &deployments, err
}

// DeleteDeployment deletes kubernetes deployment
func (c *client) DeleteDeployment(d *Deployment) error {
	return nil
}

// CreateService creates kubernetes services
func (c *client) CreateService(s *Service) error {
	return nil
}

// GetService queries kubernetes services and returns them
func (c *client) GetService(labels map[string]string) (*ServiceList, error) {
	return nil, nil
}

// UpdateService updates kubernetes service
func (c *client) UpdateService(s *Service) error {
	return nil
}

// DeleteService deletes kubernetes service
func (c *client) DeleteService(s *Service) error {
	return nil
}

// ListServices lists kubernetes services and returns them
func (c *client) ListServices() (*ServiceList, error) {
	return nil, nil
}
