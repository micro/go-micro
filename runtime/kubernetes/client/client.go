package client

import (
	"bytes"
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
	// path to kubernetes service account token
	serviceAccountPath = "/var/run/secrets/kubernetes.io/serviceaccount"
	// ErrReadNamespace is returned when the names could not be read from service account
	ErrReadNamespace = errors.New("Could not read namespace from service account secret")
)

// Client ...
type client struct {
	opts *api.Options
}

// NewClientInCluster creates a Kubernetes client for use from within a k8s pod.
func NewClientInCluster() *client {
	host := "https://" + os.Getenv("KUBERNETES_SERVICE_HOST") + ":" + os.Getenv("KUBERNETES_SERVICE_PORT")

	s, err := os.Stat(serviceAccountPath)
	if err != nil {
		log.Fatal(err)
	}
	if s == nil || !s.IsDir() {
		log.Fatal(errors.New("service account not found"))
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

// Create creates new API object
func (c *client) Create(r *Resource) error {
	b := new(bytes.Buffer)
	if err := renderTemplate(r.Kind, b, r.Value); err != nil {
		return err
	}

	return api.NewRequest(c.opts).
		Post().
		SetHeader("Content-Type", "application/yaml").
		Resource(r.Kind).
		Body(b).
		Do().
		Error()
}

// Get queries API objects and stores the result in r
func (c *client) Get(r *Resource, labels map[string]string) error {
	return api.NewRequest(c.opts).
		Get().
		Resource(r.Kind).
		Params(&api.Params{LabelSelector: labels}).
		Do().
		Into(r.Value)
}

// Update updates API object
func (c *client) Update(r *Resource) error {
	req := api.NewRequest(c.opts).
		Patch().
		SetHeader("Content-Type", "application/strategic-merge-patch+json").
		Resource(r.Kind).
		Name(r.Name)

	switch r.Kind {
	case "service":
		req.Body(r.Value.(*Service))
	case "deployment":
		req.Body(r.Value.(*Deployment))
	default:
		return errors.New("unsupported resource")
	}

	return req.Do().Error()
}

// Delete removes API object
func (c *client) Delete(r *Resource) error {
	return api.NewRequest(c.opts).
		Delete().
		Resource(r.Kind).
		Name(r.Name).
		Do().
		Error()
}

// List lists API objects and stores the result in r
func (c *client) List(r *Resource) error {
	labels := map[string]string{
		"micro": "service",
	}
	return c.Get(r, labels)
}
