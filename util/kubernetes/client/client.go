// Package client provides an implementation of a restricted subset of kubernetes API client
package client

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/util/kubernetes/api"
)

var (
	// path to kubernetes service account token
	serviceAccountPath = "/var/run/secrets/kubernetes.io/serviceaccount"
	// ErrReadNamespace is returned when the names could not be read from service account
	ErrReadNamespace = errors.New("Could not read namespace from service account secret")
	// DefaultImage is default micro image
	DefaultImage = "micro/go-micro"
	// DefaultNamespace is the default k8s namespace
	DefaultNamespace = "default"
)

// Client ...
type client struct {
	opts *api.Options
}

// Kubernetes client
type Client interface {
	// Create creates new API resource
	Create(*Resource, ...CreateOption) error
	// Get queries API resrouces
	Get(*Resource, ...GetOption) error
	// Update patches existing API object
	Update(*Resource, ...UpdateOption) error
	// Delete deletes API resource
	Delete(*Resource, ...DeleteOption) error
	// List lists API resources
	List(*Resource, ...ListOption) error
	// Log gets log for a pod
	Log(*Resource, ...LogOption) (io.ReadCloser, error)
	// Watch for events
	Watch(*Resource, ...WatchOption) (Watcher, error)
}

// Create creates new API object
func (c *client) Create(r *Resource, opts ...CreateOption) error {
	options := CreateOptions{
		Namespace: c.opts.Namespace,
	}
	for _, o := range opts {
		o(&options)
	}

	b := new(bytes.Buffer)
	if err := renderTemplate(r.Kind, b, r.Value); err != nil {
		return err
	}

	return api.NewRequest(c.opts).
		Post().
		SetHeader("Content-Type", "application/yaml").
		Namespace(options.Namespace).
		Resource(r.Kind).
		Body(b).
		Do().
		Error()
}

var (
	nameRegex = regexp.MustCompile("[^a-zA-Z0-9]+")
)

// SerializeResourceName removes all spacial chars from a string so it
// can be used as a k8s resource name
func SerializeResourceName(ns string) string {
	return nameRegex.ReplaceAllString(ns, "-")
}

// Get queries API objects and stores the result in r
func (c *client) Get(r *Resource, opts ...GetOption) error {
	options := GetOptions{
		Namespace: c.opts.Namespace,
	}
	for _, o := range opts {
		o(&options)
	}

	return api.NewRequest(c.opts).
		Get().
		Resource(r.Kind).
		Namespace(options.Namespace).
		Params(&api.Params{LabelSelector: options.Labels}).
		Do().
		Into(r.Value)
}

// Log returns logs for a pod
func (c *client) Log(r *Resource, opts ...LogOption) (io.ReadCloser, error) {
	options := LogOptions{
		Namespace: c.opts.Namespace,
	}
	for _, o := range opts {
		o(&options)
	}

	req := api.NewRequest(c.opts).
		Get().
		Resource(r.Kind).
		SubResource("log").
		Name(r.Name).
		Namespace(options.Namespace)

	if options.Params != nil {
		req.Params(&api.Params{Additional: options.Params})
	}

	resp, err := req.Raw()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, errors.New(resp.Request.URL.String() + ": " + resp.Status)
	}
	return resp.Body, nil
}

// Update updates API object
func (c *client) Update(r *Resource, opts ...UpdateOption) error {
	options := UpdateOptions{
		Namespace: c.opts.Namespace,
	}
	for _, o := range opts {
		o(&options)
	}

	req := api.NewRequest(c.opts).
		Patch().
		SetHeader("Content-Type", "application/strategic-merge-patch+json").
		Resource(r.Kind).
		Name(r.Name).
		Namespace(options.Namespace)

	switch r.Kind {
	case "service":
		req.Body(r.Value.(*Service))
	case "deployment":
		req.Body(r.Value.(*Deployment))
	case "pod":
		req.Body(r.Value.(*Pod))
	default:
		return errors.New("unsupported resource")
	}

	return req.Do().Error()
}

// Delete removes API object
func (c *client) Delete(r *Resource, opts ...DeleteOption) error {
	options := DeleteOptions{
		Namespace: c.opts.Namespace,
	}
	for _, o := range opts {
		o(&options)
	}

	return api.NewRequest(c.opts).
		Delete().
		Resource(r.Kind).
		Name(r.Name).
		Namespace(options.Namespace).
		Do().
		Error()
}

// List lists API objects and stores the result in r
func (c *client) List(r *Resource, opts ...ListOption) error {
	options := ListOptions{
		Namespace: c.opts.Namespace,
	}
	for _, o := range opts {
		o(&options)
	}

	return c.Get(r, GetNamespace(options.Namespace))
}

// Watch returns an event stream
func (c *client) Watch(r *Resource, opts ...WatchOption) (Watcher, error) {
	options := WatchOptions{
		Namespace: c.opts.Namespace,
	}
	for _, o := range opts {
		o(&options)
	}

	// set the watch param
	params := &api.Params{Additional: map[string]string{
		"watch": "true",
	}}

	// get options params
	if options.Params != nil {
		for k, v := range options.Params {
			params.Additional[k] = v
		}
	}

	req := api.NewRequest(c.opts).
		Get().
		Resource(r.Kind).
		Name(r.Name).
		Namespace(options.Namespace).
		Params(params)

	return newWatcher(req)
}

// NewService returns default micro kubernetes service definition
func NewService(name, version, typ, namespace string) *Service {
	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("kubernetes default service: name: %s, version: %s", name, version)
	}

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

	if len(namespace) == 0 {
		namespace = DefaultNamespace
	}

	Metadata := &Metadata{
		Name:      svcName,
		Namespace: SerializeResourceName(namespace),
		Version:   version,
		Labels:    Labels,
	}

	Spec := &ServiceSpec{
		Type:     "ClusterIP",
		Selector: Labels,
		Ports: []ServicePort{{
			"service-port", 8080, "",
		}},
	}

	return &Service{
		Metadata: Metadata,
		Spec:     Spec,
	}
}

// NewService returns default micro kubernetes deployment definition
func NewDeployment(name, version, typ, namespace string) *Deployment {
	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("kubernetes default deployment: name: %s, version: %s", name, version)
	}

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

	if len(namespace) == 0 {
		namespace = DefaultNamespace
	}

	Metadata := &Metadata{
		Name:        depName,
		Namespace:   SerializeResourceName(namespace),
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
				ServiceAccountName: namespace,
				Containers: []Container{{
					Name:    name,
					Image:   DefaultImage,
					Env:     []EnvVar{env},
					Command: []string{"go", "run", "."},
					Ports: []ContainerPort{{
						Name:          "service-port",
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

// NewLocalClient returns a client that can be used with `kubectl proxy`
func NewLocalClient(hosts ...string) *client {
	if len(hosts) == 0 {
		hosts[0] = "http://localhost:8001"
	}
	return &client{
		opts: &api.Options{
			Client:    http.DefaultClient,
			Host:      hosts[0],
			Namespace: "default",
		},
	}
}

// NewClusterClient creates a Kubernetes client for use from within a k8s pod.
func NewClusterClient() *client {
	host := "https://" + os.Getenv("KUBERNETES_SERVICE_HOST") + ":" + os.Getenv("KUBERNETES_SERVICE_PORT")

	s, err := os.Stat(serviceAccountPath)
	if err != nil {
		logger.Fatal(err)
	}
	if s == nil || !s.IsDir() {
		logger.Fatal(errors.New("service account not found"))
	}

	token, err := ioutil.ReadFile(path.Join(serviceAccountPath, "token"))
	if err != nil {
		logger.Fatal(err)
	}
	t := string(token)

	crt, err := CertPoolFromFile(path.Join(serviceAccountPath, "ca.crt"))
	if err != nil {
		logger.Fatal(err)
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
			BearerToken: &t,
			Namespace:   DefaultNamespace,
		},
	}
}
