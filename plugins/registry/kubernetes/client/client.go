package client

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	log "github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/plugins/registry/kubernetes/v3/client/api"
	"github.com/asim/go-micro/plugins/registry/kubernetes/v3/client/watch"
)

var (
	serviceAccountPath = "/var/run/secrets/kubernetes.io/serviceaccount"

	ErrReadNamespace = errors.New("Could not read namespace from service account secret")
)

// Client ...
type client struct {
	opts *api.Options
}

// ListPods ...
func (c *client) ListPods(labels map[string]string) (*PodList, error) {
	var pods PodList
	err := api.NewRequest(c.opts).Get().Resource("pods").Params(&api.Params{LabelSelector: labels}).Do().Into(&pods)
	return &pods, err
}

// UpdatePod ...
func (c *client) UpdatePod(name string, p *Pod) (*Pod, error) {
	var pod Pod
	err := api.NewRequest(c.opts).Patch().Resource("pods").Name(name).Body(p).Do().Into(&pod)
	return &pod, err
}

// WatchPods ...
func (c *client) WatchPods(labels map[string]string) (watch.Watch, error) {
	return api.NewRequest(c.opts).Get().Resource("pods").Params(&api.Params{LabelSelector: labels}).Watch()
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

// NewClientByHost sets up a client by host
func NewClientByHost(host string) Kubernetes {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
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

// NewClientInCluster should work similarily to the official api
// NewInClient by setting up a client configuration for use within
// a k8s pod.
func NewClientInCluster() Kubernetes {
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
