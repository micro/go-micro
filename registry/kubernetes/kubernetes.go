// Package kubernetes provides a kubernetes registry
package kubernetes

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"go-micro.dev/v6/registry"

	"github.com/pkg/errors"

	"go-micro.dev/v6/registry/kubernetes/client"
)

type kregistry struct {
	client  client.Kubernetes
	timeout time.Duration
	options registry.Options
}

var (
	// used on pods as labels & services to select
	// eg: svcSelectorPrefix+"svc.name"
	svcSelectorPrefix = "micro.mu/selector-"
	svcSelectorValue  = "service"

	labelTypeKey          = "micro.mu/type"
	labelTypeValueService = "service"

	// used on k8s services to scope a serialized
	// micro service by pod name.
	annotationServiceKeyPrefix = "micro.mu/service-"

	// Pod status.
	podRunning = "Running"

	// label name regex.
	labelRe = regexp.MustCompilePOSIX("[-A-Za-z0-9_.]")
)

// Err are all package errors.
var (
	ErrNoHostname   = errors.New("failed to get podname from HOSTNAME variable")
	ErrNoNodesFound = errors.New("you must provide at least one node")
)

// podSelector.
var podSelector = map[string]string{
	labelTypeKey: labelTypeValueService,
}

func configure(k *kregistry, opts ...registry.Option) error {
	for _, o := range opts {
		o(&k.options)
	}

	// get first host
	var host string
	if len(k.options.Addrs) > 0 && len(k.options.Addrs[0]) > 0 {
		host = k.options.Addrs[0]
	}

	if k.options.Timeout == 0 {
		k.options.Timeout = time.Second * 1
	}

	// if no hosts setup, assume InCluster
	var c client.Kubernetes
	if len(host) == 0 {
		c = client.NewClientInCluster()
	} else {
		c = client.NewClientByHost(host)
	}

	k.client = c
	k.timeout = k.options.Timeout

	return nil
}

// serviceName generates a valid service name for k8s labels.
func serviceName(name string) string {
	aname := make([]byte, len(name))

	for i, r := range []byte(name) {
		if !labelRe.Match([]byte{r}) {
			aname[i] = '_'
			continue
		}

		aname[i] = r
	}

	return string(aname)
}

// Init allows reconfig of options.
func (c *kregistry) Init(opts ...registry.Option) error {
	return configure(c, opts...)
}

// Options returns the registry Options.
func (c *kregistry) Options() registry.Options {
	return c.options
}

// Register sets a service selector label and an annotation with a
// serialized version of the service passed in.
func (c *kregistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	if len(s.Nodes) == 0 {
		return ErrNoNodesFound
	}

	svcName := s.Name

	// TODO: grab podname from somewhere better than this.
	podName, err := getPodName()
	if err != nil {
		return errors.Wrap(err, "failed to register")
	}

	// encode micro service
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}

	svc := string(b)

	pod := &client.Pod{
		Metadata: &client.Meta{
			Labels: map[string]*string{
				labelTypeKey:                             &labelTypeValueService,
				svcSelectorPrefix + serviceName(svcName): &svcSelectorValue,
			},
			Annotations: map[string]*string{
				annotationServiceKeyPrefix + serviceName(svcName): &svc,
			},
		},
	}

	if _, err := c.client.UpdatePod(podName, pod); err != nil {
		return err
	}

	return nil
}

// Deregister nils out any things set in Register.
func (c *kregistry) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {
	if len(s.Nodes) == 0 {
		return ErrNoNodesFound
	}

	svcName := s.Name

	// TODO: grab podname from somewhere better than env var.
	podName, err := getPodName()
	if err != nil {
		return errors.Wrap(err, "failed to deregister")
	}

	pod := &client.Pod{
		Metadata: &client.Meta{
			Labels: map[string]*string{
				svcSelectorPrefix + serviceName(svcName): nil,
			},
			Annotations: map[string]*string{
				annotationServiceKeyPrefix + serviceName(svcName): nil,
			},
		},
	}

	if _, err := c.client.UpdatePod(podName, pod); err != nil {
		return err
	}

	return nil
}

// GetService will get all the pods with the given service selector,
// and build services from the annotations.
func (c *kregistry) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	pods, err := c.client.ListPods(map[string]string{
		svcSelectorPrefix + serviceName(name): svcSelectorValue,
	})
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return nil, registry.ErrNotFound
	}

	// svcs mapped by version
	svcs := make(map[string]*registry.Service)

	// loop through items
	for _, pod := range pods.Items {
		if pod.Status.Phase != podRunning || pod.Metadata.DeletionTimestamp != "" {
			continue
		}
		// get serialized service from annotation
		svcStr, ok := pod.Metadata.Annotations[annotationServiceKeyPrefix+serviceName(name)]
		if !ok {
			continue
		}

		var svc registry.Service

		// unmarshal service string
		err := json.Unmarshal([]byte(*svcStr), &svc)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal service '%s' from pod annotation", name)
		}

		// merge up pod service & ip with versioned service.
		vs, ok := svcs[svc.Version]
		if !ok {
			svcs[svc.Version] = &svc
			continue
		}

		vs.Nodes = append(vs.Nodes, svc.Nodes...)
	}

	list := make([]*registry.Service, 0, len(svcs))
	for _, val := range svcs {
		list = append(list, val)
	}

	return list, nil
}

// ListServices will list all the service names.
func (c *kregistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	pods, err := c.client.ListPods(podSelector)
	if err != nil {
		return nil, err
	}

	// svcs mapped by name+version
	svcs := make(map[string]*registry.Service)

	for _, pod := range pods.Items {
		if pod.Status.Phase != podRunning || pod.Metadata.DeletionTimestamp != "" {
			continue
		}

		for k, v := range pod.Metadata.Annotations {
			if !strings.HasPrefix(k, annotationServiceKeyPrefix) {
				continue
			}

			// we have to unmarshal the annotation itself since the
			// key is encoded to match the regex restriction.
			var svc registry.Service
			if err := json.Unmarshal([]byte(*v), &svc); err != nil {
				continue
			}

			s, ok := svcs[svc.Name+svc.Version]
			if !ok {
				svcs[svc.Name+svc.Version] = &svc
				continue
			}

			// append to service:version nodes
			s.Nodes = append(s.Nodes, svc.Nodes...)
		}
	}

	i := 0
	list := make([]*registry.Service, len(svcs))

	for _, s := range svcs {
		list[i] = s
		i++
	}

	return list, nil
}

// Watch returns a kubernetes watcher.
func (c *kregistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	return newWatcher(c, opts...)
}

func (c *kregistry) String() string {
	return "kubernetes"
}

// NewRegistry creates a kubernetes registry.
func NewRegistry(opts ...registry.Option) registry.Registry {
	k := &kregistry{
		options: registry.Options{},
	}

	//nolint:errcheck,gosec
	configure(k, opts...)

	return k
}

func getPodName() (string, error) {
	podName := os.Getenv("HOSTNAME")
	if len(podName) == 0 {
		return "", ErrNoHostname
	}

	return podName, nil
}
