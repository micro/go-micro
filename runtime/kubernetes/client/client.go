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
func (c *client) Create(r interface{}) error {
	attr, err := getResourceAttrs(r)
	if err != nil {
		return err
	}

	b := new(bytes.Buffer)
	if err := renderTemplate(templates[attr.kind], b, r); err != nil {
		return err
	}

	return api.NewRequest(c.opts).
		Post().
		SetHeader("Content-Type", "application/yaml").
		Resource(attr.kind).
		Body(b).
		Do().
		Error()
}

// Get queries API objects and stores the result in r
func (c *client) Get(r interface{}, labels map[string]string) error {
	attr, err := getResourceAttrs(r)
	if err != nil {
		return err
	}

	err = api.NewRequest(c.opts).
		Get().
		Resource(attr.kind).
		Params(&api.Params{LabelSelector: labels}).
		Do().
		Into(r)

	return err
}

// Update updates API object
func (c *client) Update(r interface{}) error {
	attr, err := getResourceAttrs(r)
	if err != nil {
		return err
	}

	req := api.NewRequest(c.opts).
		Patch().
		SetHeader("Content-Type", "application/strategic-merge-patch+json").
		Resource(attr.kind).
		Name(attr.name)

	switch attr.kind {
	case "services":
		req.Body(r.(*Service).Spec)
	case "deployments":
		req.Body(r.(*Deployment).Spec)
	}

	return req.Do().Error()
}

// Delete removes API object
func (c *client) Delete(r interface{}) error {
	attr, err := getResourceAttrs(r)
	if err != nil {
		return err
	}

	return api.NewRequest(c.opts).
		Delete().
		Resource(attr.kind).
		Name(attr.name).
		Do().
		Error()
}

// List lists API objects and stores the result in r
func (c *client) List(r interface{}) error {
	attr, err := getResourceAttrs(r)
	if err != nil {
		return err
	}

	labels := map[string]string{
		"micro": "service",
	}

	err = api.NewRequest(c.opts).
		Get().
		Resource(attr.kind).
		Params(&api.Params{LabelSelector: labels}).
		Do().
		Into(r)

	return err
}
