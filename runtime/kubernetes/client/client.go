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

// UpdateDeployment
func (c *client) UpdateDeployment(name string, body interface{}) error {
	return api.NewRequest(c.opts).
		Patch().
		Resource("deployments").
		Name(name).
		Body(body).
		Do().
		Error()
}

// ListDeployments
func (c *client) ListDeployments() (*DeploymentList, error) {
	return nil, nil
}
