// Package client is the kubernetes registry client.
package client

import (
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"path"

	"go-micro.dev/v6/logger"

	"go-micro.dev/v6/registry/kubernetes/client/api"
	"go-micro.dev/v6/registry/kubernetes/client/watch"
)

var (
	serviceAccountPath = "/var/run/secrets/kubernetes.io/serviceaccount"

	// ErrReadNamespace error when failed to read namespace.
	ErrReadNamespace = errors.New("could not read namespace from service account secret")
)

// Client ...
type client struct {
	opts *api.Options
}

// NewClientByHost sets up a client by host.
func NewClientByHost(host string) Kubernetes {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			//nolint:gosec
			InsecureSkipVerify: true,
		},
		DisableCompression: true,
	}

	c := &http.Client{
		Transport: tr,
	}

	return &client{
		opts: &api.Options{
			Client:    c,
			Host:      host,
			Namespace: "default",
		},
	}
}

// NewClientInCluster should work similarly to the official api
// NewInClient by setting up a client configuration for use within
// a k8s pod.
func NewClientInCluster() Kubernetes {
	host := "https://" + os.Getenv("KUBERNETES_SERVICE_HOST") + ":" + os.Getenv("KUBERNETES_SERVICE_PORT")

	s, err := os.Stat(serviceAccountPath)
	if err != nil {
		logger.Fatal(err)
	}

	if s == nil || !s.IsDir() {
		logger.Fatal(errors.New("no k8s service account found"))
	}

	t, err := os.ReadFile(path.Join(serviceAccountPath, "token"))
	if err != nil {
		logger.Fatal(err)
	}

	token := string(t)

	ns, err := detectNamespace()
	if err != nil {
		logger.Fatal(err)
	}

	crt, err := CertPoolFromFile(path.Join(serviceAccountPath, "ca.crt"))
	if err != nil {
		logger.Fatal(err)
	}

	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    crt,
				MinVersion: tls.VersionTLS12,
			},
			DisableCompression: true,
		},
	}

	return &client{
		opts: &api.Options{
			Client:      c,
			Host:        host,
			Namespace:   ns,
			BearerToken: &token,
		},
	}
}

// ListPods ...
func (c *client) ListPods(labels map[string]string) (*PodList, error) {
	var pods PodList
	err := api.NewRequest(c.opts).Get().Resource("pods").Params(&api.Params{LabelSelector: labels}).Do().Decode(&pods)

	return &pods, err
}

// UpdatePod ...
func (c *client) UpdatePod(name string, p *Pod) (*Pod, error) {
	var pod Pod
	err := api.NewRequest(c.opts).Patch().Resource("pods").Name(name).Body(p).Do().Decode(&pod)

	return &pod, err
}

// WatchPods ...
func (c *client) WatchPods(labels map[string]string) (watch.Watch, error) {
	return api.NewRequest(c.opts).Get().Resource("pods").Params(&api.Params{LabelSelector: labels}).Watch()
}

func detectNamespace() (string, error) {
	nsPath := path.Join(serviceAccountPath, "namespace")

	// Make sure it's a file and we can read it
	if s, err := os.Stat(nsPath); err != nil {
		return "", err
	} else if s.IsDir() {
		return "", ErrReadNamespace
	}

	// Read the file, and cast to a string
	ns, err := os.ReadFile(path.Clean(nsPath))
	if err != nil {
		return string(ns), err
	}

	return string(ns), nil
}
